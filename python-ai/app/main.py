import asyncio
import json
import logging
import os
import random
import time
from typing import List
from uuid import uuid4

from fastapi import FastAPI, Request
from opentelemetry import propagate, trace
from opentelemetry.context import attach, detach
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.trace import SpanKind, Status, StatusCode
from pydantic import BaseModel


logging.basicConfig(level=logging.INFO, format="%(message)s")
logger = logging.getLogger("python-ai")


def setup_tracing() -> None:
    if os.getenv("DISABLE_OTEL_TRACING", "").lower() in {"1", "true", "yes"}:
        return

    resource = Resource.create(
        {"service.name": os.getenv("OTEL_SERVICE_NAME", "python-ai")}
    )
    provider = TracerProvider(resource=resource)

    exporter = OTLPSpanExporter(
        endpoint=os.getenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://tempo:4317"),
        insecure=True,
    )
    provider.add_span_processor(BatchSpanProcessor(exporter))
    trace.set_tracer_provider(provider)


setup_tracing()
tracer = trace.get_tracer("python-ai")
app = FastAPI(title="python-ai")


class ChatRequest(BaseModel):
    message: str


class ChatResponse(BaseModel):
    reply: str
    model: str


class EmbedRequest(BaseModel):
    text: str


class EmbedResponse(BaseModel):
    vector: List[float]


def current_trace_fields() -> dict:
    span_context = trace.get_current_span().get_span_context()
    if not span_context.is_valid:
        return {}

    return {
        "trace_id": format(span_context.trace_id, "032x"),
        "span_id": format(span_context.span_id, "016x"),
    }


def log_event(event: str, **fields) -> None:
    record = {
        "service": "python-ai",
        "event": event,
        **current_trace_fields(),
        **fields,
    }
    logger.info(json.dumps(record, ensure_ascii=False))


def get_fixed_delay_seconds() -> float:
    value = os.getenv("AI_FIXED_DELAY_SECONDS", "0")
    try:
        return float(value)
    except ValueError:
        return 0.0


@app.middleware("http")
async def tracing_and_logging_middleware(request: Request, call_next):
    extracted_context = propagate.extract(request.headers)
    token = attach(extracted_context)

    request_id = request.headers.get("X-Request-ID") or str(uuid4())
    span_name = f"{request.method} {request.url.path}"

    start = time.perf_counter()

    try:
        with tracer.start_as_current_span(span_name, kind=SpanKind.SERVER) as span:
            span.set_attribute("http.method", request.method)
            span.set_attribute("http.target", request.url.path)
            span.set_attribute("request.id", request_id)

            request.state.request_id = request_id

            response = await call_next(request)
            duration_ms = round((time.perf_counter() - start) * 1000, 2)

            span.set_attribute("http.status_code", response.status_code)
            span.set_attribute("http.duration_ms", duration_ms)

            if response.status_code >= 500:
                span.set_status(Status(StatusCode.ERROR))

            trace_fields = current_trace_fields()
            trace_id = trace_fields.get("trace_id", "")

            response.headers["X-Request-ID"] = request_id
            if trace_id:
                response.headers["X-Trace-ID"] = trace_id

            log_event(
                "http_request_completed",
                request_id=request_id,
                method=request.method,
                path=request.url.path,
                status=response.status_code,
                duration_ms=duration_ms,
            )
            return response
    except Exception as exc:
        span = trace.get_current_span()
        if span is not None:
            span.record_exception(exc)
            span.set_status(Status(StatusCode.ERROR, str(exc)))
        raise
    finally:
        detach(token)


@app.get("/health")
async def health() -> dict:
    return {"status": "ok"}


@app.post("/chat", response_model=ChatResponse)
async def chat(payload: ChatRequest, request: Request) -> ChatResponse:
    delay_seconds = get_fixed_delay_seconds()
    if delay_seconds > 0:
        await asyncio.sleep(delay_seconds)

    log_event(
        "chat_processed",
        request_id=request.state.request_id,
        model="mock-fastapi-v1",
        message_length=len(payload.message),
    )

    return ChatResponse(
        reply=f"python-ai received: {payload.message}",
        model="mock-fastapi-v1",
    )


@app.post("/embed", response_model=EmbedResponse)
async def embed(payload: EmbedRequest, request: Request) -> EmbedResponse:
    random.seed(payload.text)
    vector = [round(random.random(), 6) for _ in range(8)]

    log_event(
        "embed_processed",
        request_id=request.state.request_id,
        text_length=len(payload.text),
    )

    return EmbedResponse(vector=vector)

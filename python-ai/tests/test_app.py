from fastapi.testclient import TestClient

from app.main import app


client = TestClient(app)


def test_health():
    response = client.get("/health")
    assert response.status_code == 200
    assert response.json() == {"status": "ok"}


def test_chat():
    response = client.post("/chat", json={"message": "hello ci"})

    assert response.status_code == 200

    data = response.json()
    assert data["model"] == "mock-fastapi-v1"
    assert data["reply"] == "python-ai received: hello ci"


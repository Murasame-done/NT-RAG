-- +goose Up
-- 1. 会话表
CREATE TABLE sessions (
	id VARCHAR(36) PRIMARY KEY,
	user_id VARCHAR(255) NOT NULL,
	character_id VARCHAR(255) NOT NULL,
	title TEXT,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 2. 消息表
CREATE TABLE messages (
	id VARCHAR(36) PRIMARY KEY,
	session_id VARCHAR(36) NOT NULL,
	role VARCHAR(50) NOT NULL,
	content TEXT NOT NULL,
	trace_id VARCHAR(100),
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 3. 状态快照表（L2）
CREATE TABLE state_snapshots(
	id VARCHAR(36) PRIMARY KEY,
	session_id VARCHAR(36) NOT NULL,
	version INT NOT NULL,
	character_state JSON,
    	relationship_state JSON,
    	status VARCHAR(20),
    	source_trace_id VARCHAR(100),
    	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 4. 长期记忆条目
CREATE TABLE memory_records (
	id VARCHAR(36) PRIMARY KEY,
	session_id VARCHAR(36) NOT NULL,
	memory_type VARCHAR(50),
	summary TEXT,
	entities JSON,
	events JSON,
	confidence FLOAT,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 5. 任务表
CREATE TABLE jobs (
	id VARCHAR(36) PRIMARY KEY,
	job_type VARCHAR(50),
	play JSON,
	status VARCHAR(20) DEFAULT 'pending',
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	update_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);




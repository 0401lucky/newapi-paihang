-- 模拟 NewAPI 表结构（只含本项目用到的字段）
CREATE TABLE IF NOT EXISTS users (
  id            INT PRIMARY KEY,
  username      VARCHAR(64) NOT NULL,
  display_name  VARCHAR(64) DEFAULT '',
  role          INT DEFAULT 1,
  status        INT DEFAULT 1,
  email         VARCHAR(128) DEFAULT '',
  quota         BIGINT DEFAULT 0,
  used_quota    BIGINT DEFAULT 0,
  request_count INT DEFAULT 0,
  `group`       VARCHAR(64) DEFAULT 'default',
  aff_code      VARCHAR(32) DEFAULT '',
  aff_count     INT DEFAULT 0,
  aff_quota     BIGINT DEFAULT 0,
  aff_history   BIGINT DEFAULT 0,
  inviter_id    INT DEFAULT 0,
  created_at    BIGINT NOT NULL,
  last_login_at BIGINT DEFAULT 0,
  deleted_at    DATETIME NULL,
  INDEX idx_status_deleted (status, deleted_at)
);

CREATE TABLE IF NOT EXISTS logs (
  id                  BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id             INT NOT NULL,
  created_at          BIGINT NOT NULL,
  type                INT NOT NULL,
  content             TEXT,
  username            VARCHAR(64),
  token_name          VARCHAR(64),
  model_name          VARCHAR(128),
  quota               BIGINT DEFAULT 0,
  prompt_tokens       INT DEFAULT 0,
  completion_tokens   INT DEFAULT 0,
  use_time            INT DEFAULT 0,
  is_stream           BOOLEAN DEFAULT FALSE,
  channel             INT,
  token_id            INT,
  `group`             VARCHAR(64),
  ip                  VARCHAR(64),
  request_id          VARCHAR(64),
  upstream_request_id VARCHAR(64),
  other               TEXT,
  INDEX idx_user (user_id),
  INDEX idx_created (created_at),
  INDEX idx_model (model_name),
  INDEX idx_created_type (created_at, type)
);

CREATE TABLE IF NOT EXISTS top_ups (
  id              INT PRIMARY KEY AUTO_INCREMENT,
  user_id         INT NOT NULL,
  amount          BIGINT,
  money           DOUBLE NOT NULL,
  trade_no        VARCHAR(64),
  payment_method  VARCHAR(32),
  payment_provider VARCHAR(32),
  create_time     BIGINT NOT NULL,
  complete_time   BIGINT,
  status          VARCHAR(32) NOT NULL
);

-- 10 个用户：1=admin（隐藏候选）、2-9=正常、10=被禁用、11=软删除
INSERT INTO users (id, username, display_name, role, status, quota, used_quota, request_count, created_at, deleted_at) VALUES
  (1,  'admin',      '管理员',         100, 1, 999999999999, 0,           0,     1700000000, NULL),
  (2,  'tycoon',     '咸鱼想躺平',     1,   1, 4216060000,   2500000000,  1200,  1700100000, NULL),
  (3,  'clude_fan',  '克吹本吹',       1,   1, 2600940000,   1800000000,  3500,  1700200000, NULL),
  (4,  'sub_clude',  '小克的奴',       1,   1, 1833700000,   1200000000,  800,   1700300000, NULL),
  (5,  'nightowl',   '夜半敲键人',     1,   1, 1054250000,   900000000,   5500,  1700400000, NULL),
  (6,  'coffee',     '咖啡续命',       1,   1, 949500000,    700000000,   2200,  1700500000, NULL),
  (7,  'gourmet',    '万物皆可尝',     1,   1, 500000000,    400000000,   1800,  1700600000, NULL),
  (8,  'plain',      '',               1,   1, 100000000,    50000000,    300,   1700700000, NULL),  -- 无 display_name
  (9,  'twin_fan',   '双子奶妈',       1,   1, 200000000,    600000000,   2400,  1700800000, NULL),
  (10, 'banned',     '已封禁',         1,   2, 99999,        0,           0,     1700900000, NULL),  -- status=2
  (11, 'deleted',    '已删除',         1,   1, 99999,        0,           0,     1701000000, '2026-01-01 00:00:00');

SET @NOW := UNIX_TIMESTAMP();

-- user 2: 大消费 + claude 重度
INSERT INTO logs (user_id, created_at, type, model_name, quota, prompt_tokens, completion_tokens) VALUES
  (2, @NOW-3600,     2, 'claude-sonnet-4',  500000, 1000, 2000),
  (2, @NOW-7200,     2, 'claude-sonnet-4',  600000, 1200, 2400),
  (2, @NOW-86400*2,  2, 'claude-sonnet-4',  700000, 1500, 3000),
  (2, @NOW-86400*5,  2, 'claude-opus-4',    900000, 800,  4000);

-- user 3: 大调用次数、纯 claude（死忠粉满足）
INSERT INTO logs (user_id, created_at, type, model_name, quota, prompt_tokens, completion_tokens)
SELECT 3, @NOW - 3600*seq, 2, 'claude-sonnet-4', 100000, 500, 1500
FROM (SELECT 1 AS seq UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9 UNION SELECT 10
      UNION SELECT 11 UNION SELECT 12 UNION SELECT 13 UNION SELECT 14 UNION SELECT 15) t;
INSERT INTO logs (user_id, created_at, type, model_name, quota, prompt_tokens, completion_tokens) VALUES
  (3, @NOW-86400, 2, 'gemini-2.5-pro', 50000, 200, 800);

-- user 4: 中等
INSERT INTO logs (user_id, created_at, type, model_name, quota, prompt_tokens, completion_tokens) VALUES
  (4, @NOW-3600,    2, 'claude-sonnet-4', 200000, 600, 1800),
  (4, @NOW-86400,   2, 'gpt-4o',          150000, 400, 1200),
  (4, @NOW-86400*3, 2, 'gpt-4o',          180000, 500, 1500);

-- user 5: 熬夜冠军 — UTC+8 凌晨 N 点 = UTC (N+16) mod 24 点
-- MySQL 默认 timezone=UTC，FROM_UNIXTIME 返回 UTC。要插入 UTC 18:00-21:00 对应 UTC+8 02:00-05:00
INSERT INTO logs (user_id, created_at, type, model_name, quota, prompt_tokens, completion_tokens) VALUES
  (5, UNIX_TIMESTAMP(DATE_FORMAT(FROM_UNIXTIME(@NOW), '%Y-%m-%d 19:00:00')) - 86400*1, 2, 'claude-sonnet-4', 100000, 300, 800),  -- UTC+8 03:00
  (5, UNIX_TIMESTAMP(DATE_FORMAT(FROM_UNIXTIME(@NOW), '%Y-%m-%d 18:00:00')) - 86400*1, 2, 'gpt-4o',          120000, 400, 900),  -- UTC+8 02:00
  (5, UNIX_TIMESTAMP(DATE_FORMAT(FROM_UNIXTIME(@NOW), '%Y-%m-%d 20:00:00')) - 86400*2, 2, 'claude-opus-4',   200000, 500, 1500), -- UTC+8 04:00
  (5, UNIX_TIMESTAMP(DATE_FORMAT(FROM_UNIXTIME(@NOW), '%Y-%m-%d 17:30:00')) - 86400*3, 2, 'gemini-2.5-pro',  150000, 400, 1200), -- UTC+8 01:30
  (5, UNIX_TIMESTAMP(DATE_FORMAT(FROM_UNIXTIME(@NOW), '%Y-%m-%d 21:00:00')) - 86400*4, 2, 'claude-sonnet-4', 100000, 300, 800);  -- UTC+8 05:00

-- user 6: 单笔王
INSERT INTO logs (user_id, created_at, type, model_name, quota, prompt_tokens, completion_tokens) VALUES
  (6, @NOW-3600,  2, 'claude-opus-4', 5000000, 10000, 20000),  -- 极大单笔
  (6, @NOW-86400, 2, 'claude-sonnet-4', 100000, 300, 800);

-- user 7: 美食家（7 种模型）
INSERT INTO logs (user_id, created_at, type, model_name, quota, prompt_tokens, completion_tokens) VALUES
  (7, @NOW-3600,    2, 'claude-sonnet-4', 100000, 200, 600),
  (7, @NOW-7200,    2, 'claude-opus-4',   200000, 400, 1200),
  (7, @NOW-10800,   2, 'gpt-4o',          150000, 300, 900),
  (7, @NOW-14400,   2, 'gpt-4o-mini',     50000,  200, 500),
  (7, @NOW-18000,   2, 'gemini-2.5-pro',  120000, 250, 750),
  (7, @NOW-21600,   2, 'gemini-2.5-flash', 30000, 150, 400),
  (7, @NOW-25200,   2, 'deepseek-chat',    40000, 200, 600);

-- user 8: 无 display_name（验证回退）
INSERT INTO logs (user_id, created_at, type, model_name, quota, prompt_tokens, completion_tokens) VALUES
  (8, @NOW-3600, 2, 'gpt-4o', 50000, 100, 300);

-- user 9: 双子粉
INSERT INTO logs (user_id, created_at, type, model_name, quota, prompt_tokens, completion_tokens)
SELECT 9, @NOW - 3600*seq, 2, 'gemini-2.5-pro', 80000, 300, 900
FROM (SELECT 1 AS seq UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 UNION SELECT 5 UNION SELECT 6
      UNION SELECT 7 UNION SELECT 8 UNION SELECT 9 UNION SELECT 10
      UNION SELECT 11 UNION SELECT 12) t;

-- user 1 (admin): 也有消费，验证隐藏
INSERT INTO logs (user_id, created_at, type, model_name, quota, prompt_tokens, completion_tokens) VALUES
  (1, @NOW-3600, 2, 'claude-sonnet-4', 999999, 5000, 10000);

-- 充值记录
INSERT INTO top_ups (user_id, amount, money, status, create_time) VALUES
  (2, 2000000000, 4000.00, 'success', UNIX_TIMESTAMP() - 86400*3),
  (3, 1000000000, 2000.00, 'success', UNIX_TIMESTAMP() - 86400*10),
  (3, 500000000,  1000.00, 'success', UNIX_TIMESTAMP() - 86400*20),
  (4, 300000000,  600.00,  'success', UNIX_TIMESTAMP() - 86400*5),
  (5, 200000000,  400.00,  'pending', UNIX_TIMESTAMP() - 86400*1);

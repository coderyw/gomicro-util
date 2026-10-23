-- KEYS[1] = 待同步 zset key
-- KEYS[2] = version hash key
--
-- ARGV[1] = now，本轮最大同步 score
-- ARGV[2] = limit，每次获取数量
-- ARGV[3] = version ttl 秒，可选；不传或 <=0 表示不设置过期

local now = ARGV[1]
local limit = tonumber(ARGV[2])
local ttl = tonumber(ARGV[3] or "0")

local members = redis.call("ZRANGEBYSCORE", KEYS[1], "-inf", now, "LIMIT", 0, limit)

if #members == 0 then
    return {}
end

local result = {}

for _, member in ipairs(members) do
    local version = redis.call("HINCRBY", KEYS[2], member, 1)

    table.insert(result, member)
    table.insert(result, tostring(version))
end

if ttl and ttl > 0 then
    redis.call("EXPIRE", KEYS[2], ttl)
end

return result
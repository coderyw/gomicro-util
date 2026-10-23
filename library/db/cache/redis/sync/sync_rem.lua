-- KEYS[1] = 待同步 zset key
-- KEYS[2] = version hash key
--
-- ARGV[1] = member
-- ARGV[2] = version

local member = ARGV[1]
local version = ARGV[2]

local currentVersion = redis.call("HGET", KEYS[2], member)

if currentVersion == version then
    redis.call("ZREM", KEYS[1], member)
    redis.call("HDEL", KEYS[2], member)
    return 1
end

return 0
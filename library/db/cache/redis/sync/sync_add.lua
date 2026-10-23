  -- KEYS[1] = zset key
  -- KEYS[2] = version hash key
  -- ARGV[1] = score
  -- ARGV[2] = member
  -- ARGV[3] = mode: 1 = 正常写   2 = 重试写(GT)

  local mode = tonumber(ARGV[3])

  if mode == 2 then
      -- 重试路径：score 只能向后推，确保 now+1 一定生效，跳到下一轮 cron
      redis.call("ZADD", KEYS[1], "GT", ARGV[1], ARGV[2])
  else
      -- 正常写：根据 versionKey 是否存在区分"正在被同步" vs "新入队/排队等待"
      if redis.call("HEXISTS", KEYS[2], ARGV[2]) == 1 then
          -- 该 field 正在被某次 sync 流程持有（HINCRBY 写过版本号、还没被 rem 清掉）
          -- 用普通 ZADD 覆盖 score，配合下面的 HDEL 让本次 rem 必然版本不匹配
          -- field 以新 score 留在 zset 里，等下一轮 cron 再扫
          redis.call("ZADD", KEYS[1], ARGV[1], ARGV[2])
      else
          -- 没在同步流程中（新入队 / 已在队列等待）
          -- 用 NX 只新增不覆盖，保留首次入队时间，避免频繁 PutSyncKey 把 score 一推再推
          redis.call("ZADD", KEYS[1], "NX", ARGV[1], ARGV[2])
      end
  end

  redis.call("HDEL", KEYS[2], ARGV[2])

  return 1
package tools

const Send2PendingScript = `
-- id存在 stop不存在 不允许重复投递
local msg = redis.call('ZSCORE', KEYS[6], ARGV[7])
local stop = redis.call('HGet', KEYS[9], ARGV[7])
if msg and not stop then return end

redis.call('HDel', KEYS[9], ARGV[7])  -- del stop
redis.call('HSet', KEYS[1], ARGV[7], ARGV[1])  -- info hset
redis.call('HSet', KEYS[3], ARGV[7], ARGV[3]) -- ttl hset
redis.call('HSet', KEYS[4], ARGV[7], ARGV[4]) -- retry count hset
redis.call('HSet', KEYS[5], ARGV[7], ARGV[5]) -- bucketName hset
redis.call('ZAdd', KEYS[6], ARGV[6], ARGV[7]) -- pending zset
redis.call('HSet', KEYS[8], ARGV[7], ARGV[8]) -- status hset
redis.call('HDel', KEYS[10], ARGV[7]) -- result hset
local next = tonumber(ARGV[2])
if (next ~= 0) then 
    redis.call('ZAdd', KEYS[2], ARGV[6], ARGV[7]) -- period zset
    redis.call('HSet', KEYS[7], ARGV[7], ARGV[4]) -- period retry hset
end
`

const Pending2ReadyScript = `
local msgs = redis.call('ZRangeByScore', KEYS[1], '0', ARGV[1])  -- get ready msg
if (#msgs == 0) then return end
local args2 = {'LPush', KEYS[2]} -- push into ready
for _,v in ipairs(msgs) do
	table.insert(args2, v) 
    -- ready和running中是否存在相同id的任务
    local ready_msg = redis.call('LPOS', KEYS[2], v)
    local running_msg = redis.call('ZSCORE', KEYS[3], v)
    if not ready_msg and not running_msg then
		redis.call('ZREM', KEYS[1], v)
		redis.call('HSet', KEYS[4], v, ARGV[2])
		redis.call('HDel',  KEYS[5], v)
		if (#args2 == 4000) then
			redis.call(unpack(args2))
			args2 = {'LPush', KEYS[2]}
		end
    end
end
-- 这里的2是因为本身里面有两个元素
if (#args2 > 2) then 
	redis.call(unpack(args2))
end
-- redis.call('ZRemRangeByScore', KEYS[1], '0', ARGV[1])  -- remove msgs from pending
`

const Ready2RunningScript = `
local msg = redis.call('RPop', KEYS[1])
if (not msg) then return end
local msgTTL = redis.call('HGet', KEYS[3], msg)
if (not msgTTL) then return end
local num = tonumber(msgTTL)
local current_time = tonumber(redis.call('TIME')[1])
redis.call('ZAdd', KEYS[2], num + current_time, msg)
redis.call('HSet', KEYS[4], msg, ARGV[1])
redis.call('HDel', KEYS[5], msg)
return msg
`

// Running2PendingScript 周期任务生成下一次任务
const Running2PendingScript = `
local msg = redis.call('ZSCORE', KEYS[1], ARGV[2]) -- get next time
if (not msg) then return end
local next = tonumber(msg)
local now = tonumber(ARGV[1])
if (next > now) then return end
local pmsg = redis.call('ZSCORE', KEYS[2], ARGV[2]) -- get pending msg 
if (pmsg) then return end  -- pending exist return
redis.call('ZAdd', KEYS[1], ARGV[3], ARGV[2])
redis.call('ZAdd', KEYS[2], ARGV[3], ARGV[2])
`

// Running2ReadyScript 超时重试
const Running2ReadyScript = `
local running2ready = function(msgs)
	local retryCounts = redis.call('HMGet', KEYS[2], unpack(msgs)) -- get retry count
	for i,v in ipairs(retryCounts) do
		local k = msgs[i]
        redis.call('ZREM', KEYS[1], k)
        -- 判断是否是周期任务，当前+ttl是否小于next time
        local msg = redis.call('ZSCORE', KEYS[6], k) -- get next time
        if (msg) then 
            -- 周期任务
            local current_time = tonumber(redis.call('TIME')[1])
            local ttl = tonumber(redis.call('HGet', KEYS[4], k))
            local next = tonumber(msg)
            if current_time + ttl < next then
                if v ~= false and v ~= nil and v ~= '' and tonumber(v) > 0 then
			        redis.call("HIncrBy", KEYS[2], k, -1) -- reduce retry count
			        redis.call("LPush", KEYS[3], k) -- add to ready
                    redis.call("HSet", KEYS[7], k, ARGV[2])  -- 超时
                else
                    local count = tonumber(redis.call('HGet', KEYS[8], k))
                    redis.call('HSet', KEYS[2], k, count)
                    redis.call("HSet", KEYS[5], k, 'TimeOut')  -- 超时
                    redis.call("HSet", KEYS[7], k, ARGV[3])  -- 错误
                end
            else
                -- 获取周期任务的retry count key,然后将retry重置
                local count = tonumber(redis.call('HGet', KEYS[8], k))
                redis.call('HSet', KEYS[2], k, count)
                redis.call("HSet", KEYS[7], k, ARGV[3])  -- 超时
            end
        else
            -- 非周期任务
            if v ~= false and v ~= nil and v ~= '' and tonumber(v) > 0 then
			    redis.call("HIncrBy", KEYS[2], k, -1) -- reduce retry count
			    redis.call("LPush", KEYS[3], k) -- add to ready
                redis.call("HSet", KEYS[7], k, ARGV[2])  -- 超时
		    else
                redis.call("HSet", KEYS[5], k, 'TimeOut')  -- 超时
                redis.call("HSet", KEYS[7], k, ARGV[3])  -- 超时
			    redis.call("HDel", KEYS[2], k) -- del retry count
			    redis.call("HDel", KEYS[4], k) -- del ttl
		    end
        end
	end
end

-- running中的是执行时间+ttl
local msgs = redis.call('ZRangeByScore', KEYS[1], '0', ARGV[1])  -- get retry msg
if (#msgs == 0) then return end
if #msgs < 4000 then
	running2ready(msgs)
else
	local buf = {}
	for _,v in ipairs(msgs) do
		table.insert(buf, v)
		if #buf == 4000 then
			running2ready(buf)
			buf = {}
		end
	end
	if (#buf > 0) then
		running2ready(buf)
	end
end
-- redis.call('ZRemRangeByScore', KEYS[1], '0', ARGV[1])  -- remove msgs from running
`

const Running2FinishScript = `
redis.call('HSet', KEYS[1], ARGV[1], ARGV[2])
redis.call('HSet', KEYS[2], ARGV[1], ARGV[3])
redis.call('ZREM', KEYS[3], ARGV[1]) -- del running
local msg = redis.call('ZSCORE', KEYS[6], ARGV[1]) -- judge period
if (not msg) then
    redis.call("HDel", KEYS[4], ARGV[1]) -- del retry count
    redis.call("HDel", KEYS[5], ARGV[1]) -- del ttl
end
`

const TerminateScript = `
-- 删除pending/ready/running中的消息
redis.call('ZREM', KEYS[1], ARGV[1])
redis.call('LRem', KEYS[2], 0, ARGV[1])
-- 设置stop信号
redis.call('HSet', KEYS[3], ARGV[1], 1)
`

const DeleteScript = `
-- 终止脚本
-- 删除pending/ready/running中的消息
redis.call('ZREM', KEYS[1], ARGV[1])
redis.call('ZREM', KEYS[2], ARGV[1])
redis.call('HDel', KEYS[4], ARGV[1])
redis.call('ZREM', KEYS[5], ARGV[1])
redis.call('HDel', KEYS[6], ARGV[1])
redis.call('LRem', KEYS[7], 0, ARGV[1])
redis.call('HDel', KEYS[8], ARGV[1])
redis.call('HDel', KEYS[9], ARGV[1])
redis.call('HDel', KEYS[10], ARGV[1])
-- 设置stop信号
redis.call('HSet', KEYS[3], ARGV[1], 1)
`

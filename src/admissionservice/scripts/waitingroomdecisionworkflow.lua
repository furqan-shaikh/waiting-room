#!lua name=waitingroomdecisionworkflow

local function isRedisError(result)
    if type(result) == "table" and result.err then
        return true
    end
    return false
end

local function getCurrentTimestamp()
    local server_time = redis.call('TIME')
    local seconds = server_time[1]
    local microseconds = server_time[2]
    return tonumber(seconds)
end

local function addUser(key, newTtl, sessionToken)
    -- Add session token to the sorted set using ZADD
    return redis.pcall("ZADD", key, newTtl, sessionToken)
end

-- invoke as: FCALL waitingroomdecisionworkflow 0 roomid1 3 sessiontoken 5
local function executeWaitingRoomWorkflow(keys, args)
    
    -- -- constants
    local MINIMUM_TIMESTAMP = "-inf"
    local DECISION_ADMIT = "admit"
    local DECISION_WAIT = "wait"

    -- arguments
    local roomId = args[1]
    local maxActiveUserCount = tonumber(args[2])
    local sessionToken = args[3]
    local ttl = tonumber(args[4])
    
    -- construct key name for storing session tokens. Key is created per wwiting room
    local key = "room:" .. roomId .. ":session_tokens"
    local decision = ""
    local currentTimestamp = getCurrentTimestamp()
    local newTtl = currentTimestamp + ttl
    local roomSize = 0

    -- Clear the sorted set using ZREMRANGEBYSCORE. This ensures that session tokens that have expired are removed and 
    -- hence allows us to determine the waiting room count without the need to maintain a separate key for count
    -- deletes all session tokens whose scores <=  current timesamp i.e expired ones
    local result = redis.pcall("ZREMRANGEBYSCORE", key, MINIMUM_TIMESTAMP, currentTimestamp)
    if isRedisError(result) then
         redis.log(redis.LOG_NOTICE, "Failed to remove scores in " .. key)
         return result.err
    end

    -- Get the session token from the sorted set.
    -- ZSCORE room:{room_id}:session_tokens <token>. If it returns a score, they are in. If it returns nil, they are a new request.
    result = redis.pcall("ZSCORE", key, sessionToken)
    if isRedisError(result) then
         redis.log(redis.LOG_NOTICE, "Failed to Get the session token from the sorted set " .. key)
         return result.err
    end

    -- Get the count of elements in Sorted Set using ZCARD
    redis.log(redis.LOG_NOTICE, "Check capacity " .. key)
    local capacityResult = redis.pcall("ZCARD", key)
    if isRedisError(capacityResult) then
        redis.log(redis.LOG_NOTICE, "Failed to check capacity for key " .. key)
        return capacityResult.err
    end
    roomSize = capacityResult

    if result then
        -- session token exists, this implies user is already in
        redis.log(redis.LOG_NOTICE, "session token exists, this implies user is already in " .. key .. " and session token " .. sessionToken)
        result = addUser(key, newTtl, sessionToken)
        if isRedisError(result) then
         redis.log(redis.LOG_NOTICE, "Failed to refresh TTL for key " .. key .. " and session token " .. sessionToken)
         return result.err
        end
        decision = DECISION_ADMIT
    else
        redis.log(redis.LOG_NOTICE, "COUNT:  " .. roomSize)
        if roomSize < maxActiveUserCount then
            result = addUser(key, newTtl, sessionToken)
            if isRedisError(result) then
                redis.log(redis.LOG_NOTICE, "Failed to add session token for key " .. key)
                return result.err
            end
            roomSize = roomSize + 1
            decision = DECISION_ADMIT
        else
            decision = DECISION_WAIT
        end
    end
    return { "decision", decision, "numberOfActiveUsers", roomSize }

end



redis.register_function('waitingroomdecisionworkflow', executeWaitingRoomWorkflow)


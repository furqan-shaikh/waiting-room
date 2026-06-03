#!lua name=waitingroomdecisionworkflow

local function isRedisError(result)
    if type(result) == "table" and result.err then
        redis.log(redis.LOG_NOTICE, result.err)
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

local function trackWaitingUser(waitingUsersKey, sessionToken, waitingTtl)

    local result = redis.pcall("HSET", waitingUsersKey, sessionToken, 1)
    if isRedisError(result) then
        redis.log(redis.LOG_NOTICE, "Failed to add user to waiting list for key " .. waitingUsersKey .. " and session token " .. sessionToken)
        return false
    end
    -- set expiry : waiting TTL in seconds
    local expiryResult = redis.pcall("HEXPIRE", waitingUsersKey, waitingTtl, "FIELDS", 1, sessionToken)
    if isRedisError(expiryResult) then
        redis.log(redis.LOG_NOTICE, "Failed to add expiry for key " .. waitingUsersKey .. " and session token " .. sessionToken)
        return false
    end
    
    return true
end

-- Assume maxActiveUsersCount = 3, activeSessionTtlInSeconds = 3600 (1 hour), roomId: "1"
-- Lets say three users are admitted. They arrive at 13:00, 13:10, 13:15 respectively. So the "room:1:session_tokens" sorted set has the following entries:
-- 	1. "token1" 14:00 (its equivalent timestamp, but we show time for easier understanding)
-- 	2. "token2" 14:10
-- 	3. "token3" 14:15

-- Lets say 4th user arrives at 13:17 and since the current count >= maxActiveUsersCount, decision is "WAIT". The logic for estimated wait time:

-- 	1. Get the lowest timestamp from sorted set using: ZRANGE room:1:session_tokens 0 0 WITHSCORES
-- 		a. In our case, this returns "token1" 14:00
-- 		b. If there are multiple, the above command still returns only 1 as our start and stop index are 0
-- 	2. Estimated waiting time: 14:00 - 13:17 i.e. 43 minutes
local function getEstimatedWaitingTime(key, sessionToken)
    local estimatedWaitingTimeInMinutes = 0
    local result = redis.pcall("ZRANGE", key, 0, 0, "WITHSCORES")
    if isRedisError(result) then
        redis.log(redis.LOG_NOTICE, "Failed to get estimated waiting time as ZRANGE failed for key " .. key .. " and session token " .. sessionToken)
        return -1
    end

    -- if no data exists, returns an empty lua table: {}
    -- if matching data exists, returns a lua table: {"token", "score"}
    if next(result) == nil then
        redis.log(redis.LOG_NOTICE, "No matching tokens by ZRANGE for key " .. key .. " and session token " .. sessionToken)
        return estimatedWaitingTimeInMinutes
    end
    local lowestTimestamp = tonumber(result[2]) -- Scores are returned as strings in Lua
    local currentTimestamp = getCurrentTimestamp()
    local diff = lowestTimestamp - currentTimestamp
    estimatedWaitingTimeInMinutes = math.ceil(diff / 60)
    redis.log(redis.LOG_NOTICE, "lowestTimestamp " .. lowestTimestamp .. " currentTimestamp " .. currentTimestamp .. " diff " .. diff .. " estimatedWaitingTimeInMinutes " .. estimatedWaitingTimeInMinutes)
    return estimatedWaitingTimeInMinutes
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
    local waitingTtl = tonumber(args[5])
    
    -- construct key name for storing session tokens. Key is created per waiting room
    local key = "room:" .. roomId .. ":session_tokens"
    local waitingUsersKey = "room:" .. roomId .. ":waiting_users"
    local decision = ""
    local currentTimestamp = getCurrentTimestamp()
    local newTtl = currentTimestamp + ttl
    local roomSize = 0
    local waitingUsersSize = 0
    local estimatedWaitingTimeInMinutes = 0

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
            -- track this waiting user
            local waitingUserStatus = trackWaitingUser(waitingUsersKey, sessionToken, waitingTtl)
            if waitingUserStatus == false then
                local errMessage = "Failed to track waiting user for key " .. waitingUsersKey .. " and session token " .. sessionToken
                redis.log(redis.LOG_NOTICE, errMessage)
                return { err = errMessage }
            end

            -- compute estimated waiting time. 
            -- Currently estimated waiting time is based on the next active session expiry.
            -- This is not a guaranteed per-user queue wait time
            estimatedWaitingTimeInMinutes = getEstimatedWaitingTime(key, sessionToken)
            decision = DECISION_WAIT
        end
    end

    -- if decision is admit, remove the user from waiting user
    if decision == DECISION_ADMIT then
        local removeWaitingUserResult = redis.pcall("HDEL", waitingUsersKey, sessionToken)
        if isRedisError(removeWaitingUserResult) then
            redis.log(redis.LOG_NOTICE, "Failed to remove session token from waiting users for key " .. waitingUsersKey .. " and session token " .. sessionToken)
        end
    end

    -- get the waiting user count
    local waitingUsersLengthResult = redis.pcall("HLEN", waitingUsersKey)
    if isRedisError(waitingUsersLengthResult) then
         redis.log(redis.LOG_NOTICE, "Failed to get size of waiting users for key " .. waitingUsersKey)
    else
        waitingUsersSize = waitingUsersLengthResult
    end
    
    return { "decision", decision, "numberOfActiveUsers", roomSize, "numberOfWaitingUsers", waitingUsersSize, "estimatedWaitingTimeInMinutes", estimatedWaitingTimeInMinutes}

end



redis.register_function('waitingroomdecisionworkflow', executeWaitingRoomWorkflow)


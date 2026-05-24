(function () {
    "use strict";

    var POLL_INTERVAL_MS = 10000;

    var statusTitle = document.getElementById("status-title");
    var statusMessage = document.getElementById("status-message");
    var roomIdLabel = document.getElementById("room-id");
    var nextCheck = document.getElementById("next-check");

    var roomId = getRoomIdFromPath(window.location.pathname);
    var pollTimer = null;
    var countdownTimer = null;

    if (!roomId) {
        showError("Waiting room unavailable", "This waiting room URL is not valid.");
        return;
    }

    roomIdLabel.textContent = roomId;
    checkStatus();

    function getRoomIdFromPath(pathname) {
        var match = pathname.match(/^\/waitingRooms\/([^/]+)\/?$/);
        return match ? decodeURIComponent(match[1]) : "";
    }

    function checkStatus() {
        clearTimers();
        setState("checking", "Checking your place", "We are checking whether your session can continue.");

        fetch(statusUrl(), {
            method: "GET",
            headers: {
                "Accept": "application/json"
            },
            cache: "no-store"
        })
            .then(function (response) {
                if (!response.ok) {
                    throw new Error("Status request failed with HTTP " + response.status);
                }
                return response.json();
            })
            .then(handleDecision)
            .catch(function (err) {
                showError("Could not check your place", "We will try again in 10 seconds.");
                scheduleNextCheck();
                if (window.console && typeof window.console.warn === "function") {
                    window.console.warn(err);
                }
            });
    }

    function statusUrl() {
        return "/waitingRooms/" + encodeURIComponent(roomId) + "/status";
    }

    function handleDecision(payload) {
        if (!payload || typeof payload.decision !== "string") {
            throw new Error("Invalid status response");
        }

        if (payload.decision === "admit") {
            setState("admitted", "You are next", "Redirecting you now.");
            redirectToOrigin(payload.origin);
            return;
        }

        if (payload.decision === "wait") {
            setState("waiting", "You are in line", "Your session is waiting for an available slot.");
            scheduleNextCheck();
            return;
        }

        throw new Error("Unknown waiting room decision: " + payload.decision);
    }

    function redirectToOrigin(origin) {
        if (!origin || typeof origin !== "string") {
            showError("Origin unavailable", "The waiting room admitted you, but no origin was provided.");
            return;
        }

        window.location.assign(origin);
    }

    function scheduleNextCheck() {
        var remainingSeconds = POLL_INTERVAL_MS / 1000;
        nextCheck.textContent = remainingSeconds + "s";

        countdownTimer = window.setInterval(function () {
            remainingSeconds -= 1;
            nextCheck.textContent = remainingSeconds > 0 ? remainingSeconds + "s" : "now";
        }, 1000);

        pollTimer = window.setTimeout(checkStatus, POLL_INTERVAL_MS);
    }

    function showError(title, message) {
        setState("error", title, message);
    }

    function setState(state, title, message) {
        document.body.dataset.state = state;
        statusTitle.textContent = title;
        statusMessage.textContent = message;
        if (state === "checking" || state === "admitted") {
            nextCheck.textContent = "now";
        }
    }

    function clearTimers() {
        if (pollTimer) {
            window.clearTimeout(pollTimer);
            pollTimer = null;
        }
        if (countdownTimer) {
            window.clearInterval(countdownTimer);
            countdownTimer = null;
        }
    }
}());

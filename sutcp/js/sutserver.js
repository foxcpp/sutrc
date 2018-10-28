/*
This file contains JS wrappers for all HTTP API operations to simplify
their usage.

All operations are async because they use AJAX.
All operations that require authorization get session token
from sutcp_session cookie (see variables below).

successCallback is called on successful operation completion
and failureCallback called on failure with error message passed 
as first argument.
*/

// Added before all API endpoints.
// Can be used to override expected HTTP API location.
// Current value uses api/ prefix relative to sutcp files location.
var apiPrefix = "api";

// Name of cookie where session token will be saved.
var cookieName = "sutcp_session"

// Initialize session using certain pass-code.
//
// On successful authorization successCallback will be called. Session
// token will be saved to sutcp_session cookie.
//
// If request fails because of invalid pass-code - invalidCredsCallback
// will be called. On other error failureCallback will be called.
function login(pass, successCallback, invalidCredsCallback, failureCallback) {
    "use strict"
    var xhr = $.ajax({
        method: "POST",
        url: apiPrefix + "/login?" + jQuery.param({token: pass}) 
    }).done(function (data) {
        Cookies.set(cookieName, data.token)
        successCallback()
    }).fail(function (resp) {
        if (resp.status == 403) {
            invalidCredsCallback()
        } else {
            failureCallback(getErrorMessage(resp))
        }
    })
}

// Terminate current session.
function logout(successCallback, failureCallback) {
    "use strict"
    var xhr = $.ajax({
        method: "POST",
        url: apiPrefix + "/logout",
        headers: {
            Authorization: Cookies.get(cookieName)
        }
    }).done(function (data) {
        successCallback()
    }).fail(function (resp) {
        failureCallback(getErrorMessage(resp))
    })
}

// Request known agents list from server.
// List will be passed to successCallback, see HTTP_API.md for object structure.
function getAgentsList(successCallback, failureCallback) {
    "use strict"
    var xhr = $.ajax({
        method: "GET",
        url: apiPrefix + "/agents",
        headers: {
            Authorization: Cookies.get(cookieName)
        }
    }).done(function (data) {
        successCallback(data.agents, data.online)
    }).fail(function (resp) {
        failureCallback(getErrorMessage(resp))
    })
    return xhr
}

// Change agent name from 'from' to 'to'.
function renameAgent(from, to, successCallback, failureCallback) {
    "use strict"
    $.ajax({
        method: "PATCH",
        url: apiPrefix + "/agents?" + jQuery.param({id: from, newId: to}),
        dataType: 'text',
        headers: {
            Authorization: Cookies.get(cookieName)
        }
    }).done(function () {
        successCallback()
    }).fail(function (resp) {
        failureCallback(getErrorMessage(resp))
    })
}

// De-register agent from server.
function deleteAgent(id, successCallback, failureCallback) {
    "use strict"
    $.ajax({
        method: "DELETE",
        url: apiPrefix + "/agents?" + jQuery.param({id: id}),
        dataType: 'text',
        headers: {
            Authorization: Cookies.get(cookieName)
        }
    }).done(function () {
        successCallback()
    }).fail(function (resp) {
        failureCallback(getErrorMessage(resp))
    })
}

// Main function for interaction with agents.
//
// target is comma-separated list of agents which should receive task.
// See sutrc's README.md for information about tasks concept and how it works.
// See HTTP_API.md for valid values of 'object' argument.
//
// You can use timeoutSecs to override default (26 seconds) task result waiting timeout.
// Result object (see HTTP_API.md for it's structure) will be passed to successCallback.
function submitTask(target, object, successCallback, failureCallback, timeoutSecs) {
    if (timeoutSecs == undefined) {
        timeoutSecs = 26
    }
    "use strict"
    var xhr = $.ajax({
        method: "POST",
        url: apiPrefix + "/tasks?" + jQuery.param({target: target, timeout: timeoutSecs}),
        data: JSON.stringify(object),
        headers: {
            Authorization: Cookies.get(cookieName)
        }
    }).done(function (data) {
        successCallback(data)
    }).fail(function (resp) {
        failureCallback(getErrorMessage(resp))
    })
    return xhr
}

// Shortcut for deletefile tasktype. Deletes file fullpath at target's filesystem.
// Doesn't works if target includes more than one agent (you need to manually use submitTask for this).
//
// Default timeout is decreased to 5 seconds to improve interface responsivness.
function deleteFile(target, fullpath, successCallback, failureCallback) {
    "use strict"
    var xhr = submitTask(target, {type: "deletefile", path: fullpath}, function (result) {
        if (result.results[0].error) {
            failureCallback(result.results[0].msg)
            return
        }
        
        successCallback()
    }, function (msg) {
        failureCallback(msg)
    }, 5)
}

// Shortcut for movetask tasktype. Moves file from frompath to topath at target's filesystem.
// Doesn't works if target includes more than one agent (you need to manually use submitTask for this).
//
// Default timeout is decreased to 5 seconds to improve interface responsivness.
function moveFile(target, frompath, topath, successCallback, failureCallback) {
    "use strict"
    var xhr = submitTask(target, {type: "movefile", frompath: frompath, topath: topath}, function (result) {
        if (result.results[0].error) {
            failureCallback(result.results[0].msg)
            return
        }
        
        successCallback()
    }, function (msg) {
        failureCallback(msg)
    }, 5)
}

// Shortcut for dircontents tasktype. Requests contents of certain directory at target's filesystem.
// Doesn't works if target includes more than one agent (you need to manually use submitTask for this).
//
// Result object structure is documented in HTTP_API.md.
//
// Default timeout is decreased to 5 seconds to improve interface responsivness.
function directoryContents(target, fullpath, successCallback, failureCallback) {
    "use strict"
    var xhr = submitTask(target, {type: "dircontents", dir: fullpath}, function (result) {
        successCallback(result.results[0].contents)
    }, function (msg) {
        failureCallback(msg)
    }, 5)
    return xhr
}

// Shortcut for downloadfile tasktype. Requests agent to upload certain file from filesystem to
// server.
//
// One-use URL to file will be passed to successCallback.
function downloadFile(target, fullpath, successCallback, failureCallback) {
    "use strict"
    var xhr = submitTask(target, {type: "uploadfile", path: fullpath}, function (result) {
        if (result.results[0].error) {
            failureCallback(result.results[0].msg)
            return
        }
        successCallback(result.results[0].url)
    }, function (msg) {
        failureCallback(msg)
    }, /*timeout*/ 240)
    return xhr
}

// Shortcut for uploadfile tasktype. Uploads file object to server and requests
// agent to download&save it to specified path (fullpath).
//
// file - object implementing Blob API.
function uploadFile(target, file, fullpath, successCallback, failureCallback) {
    "use strict"
    $.ajax({
        method: "POST",
        url: apiPrefix + "/filedrop/" + file.name,
        data: file,
        contentType: false,
        processData: false,
        headers: {
            Authorization: Cookies.get(cookieName)
        }
    }).done(function (data) {
        submitTask(target, {type: "downloadfile", url: String(data), out: fullpath}, successCallback, failureCallback)    
    }).fail(function (resp) {
        failureCallback(getErrorMessage(resp))
    })
}

// Get current value of agents self-registration switch.
function getSelfregStatus(successCallback, failureCallback) {
    var xhr = $.ajax({
        method: "GET",
        url: apiPrefix + "/agents_selfreg?",
        headers: {
            Authorization: Cookies.get(cookieName)
        }
    }).done(function (data) {
        successCallback(data == "1")
    }).fail(function (resp) {
        failureCallback(getErrorMessage(resp))
    })    
}

// Change value of agents self-registration switch.
function setSelfregStatus(val, successCallback, failureCallback) {
    var xhr = $.ajax({
        method: "POST",
        url: apiPrefix + "/agents_selfreg?" + jQuery.param({enabled: val}),
        headers: {
            Authorization: Cookies.get(cookieName)
        }
    }).done(function (data) {
        successCallback(val)
    }).fail(function (resp) {
        failureCallback(getErrorMessage(resp))
    })
}

// This is helper function which retreives error message from failed request.
//
// If response body contains JSON - 'msg' field will be used. Otherwise textual
// description of HTTP status code is returned.
function getErrorMessage(resp) {
    if (resp.responseJSON != undefined) {
        return resp.responseJSON.msg
    } else {
        return resp.statusText
    }
}
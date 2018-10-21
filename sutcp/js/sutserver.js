var apiPrefix = "api";

function login(pass, successCallback, invalidCredsCallback, failureCallback) {
    "use strict"
    var xhr = $.ajax({
        method: "POST",
        url: apiPrefix + "/login?" + jQuery.param({token: pass}) 
    }).done(function (data) {
        successCallback(data.token)
    }).fail(function (resp) {
        if (resp.status == 403) {
            invalidCredsCallback()
        } else {
            failureCallback(getErrorMessage(resp))
        }
    })
}

function logout(successCallback, failureCallback) {
    "use strict"
    var xhr = $.ajax({
        method: "POST",
        url: apiPrefix + "/logout",
        headers: {
            Authorization: token
        }
    }).done(function (data) {
        successCallback()
    }).fail(function (resp) {
        failureCallback(getErrorMessage(resp))
    })
}

function getAgentsList(successCallback, failureCallback) {
    "use strict"
    var token = Cookies.get("token")
    var xhr = $.ajax({
        method: "GET",
        url: apiPrefix + "/agents",
        headers: {
            Authorization: token
        }
    }).done(function (data) {
        successCallback(data.agents, data.online)
    }).fail(function (resp) {
        failureCallback(getErrorMessage(resp))
    })
    return xhr
}

function renameAgent(from, to, successCallback, failureCallback) {
    "use strict"
    $.ajax({
        method: "PATCH",
        url: apiPrefix + "/agents?" + jQuery.param({id: from, newId: to}),
        headers: {
            "Authorization": Cookies.get("token")
        }
    }).done(function () {
        successCallback()
    }).fail(function (resp) {
        failureCallback(getErrorMessage(resp))
    })
}

function submitTask(target, object, successCallback, failureCallback, timeout) {
    if (timeout == undefined) {
        timeout = 26
    }
    "use strict"
    var xhr = $.ajax({
        method: "POST",
        url: apiPrefix + "/tasks?" + jQuery.param({target: target, timeout: timeout}),
        data: JSON.stringify(object),
        headers: {
            Authorization: Cookies.get("token")
        }
    }).done(function (data) {
        successCallback(data)
    }).fail(function (resp) {
        failureCallback(getErrorMessage(resp))
    })
    return xhr
}

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

function directoryContents(target, fullpath, successCallback, failureCallback) {
    "use strict"
    var xhr = submitTask(target, {type: "dircontents", dir: fullpath}, function (result) {
        successCallback(result.results[0].contents)
    }, function (msg) {
        failureCallback(msg)
    })
    return xhr
}

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

function uploadFile(target, file, fullpath, successCallback, failureCallback) {
    "use strict"
    $.ajax({
        method: "POST",
        url: apiPrefix + "/filedrop/" + file.name,
        data: file,
        contentType: false,
        processData: false,
        headers: {
            Authorization: Cookies.get("token")
        }
    }).done(function (data) {
        console.log("ok to push")
        submitTask(target, {type: "downloadfile", url: String(data), out: fullpath}, successCallback, failureCallback)    
    }).fail(function (resp) {
        failureCallback(getErrorMessage(resp))
    })
}

function getSelfregStatus(successCallback, failureCallback) {
    var xhr = $.ajax({
        method: "GET",
        url: apiPrefix + "/agents_selfreg?",
        headers: {
            Authorization: Cookies.get("token")
        }
    }).done(function (data) {
        successCallback(data == "1")
    }).fail(function (resp) {
        failureCallback(getErrorMessage(resp))
    })    
}

function setSelfregStatus(val, successCallback, failureCallback) {
    var xhr = $.ajax({
        method: "POST",
        url: apiPrefix + "/agents_selfreg?" + jQuery.param({enabled: val}),
        headers: {
            Authorization: Cookies.get("token")
        }
    }).done(function (data) {
        successCallback(val)
    }).fail(function (resp) {
        failureCallback(getErrorMessage(resp))
    })
}

function getErrorMessage(resp) {
    var msg
    if (resp.responseJSON != undefined) {
        msg = resp.responseJSON.msg
    } else {
        msg = resp.statusText
    }
    return msg
}
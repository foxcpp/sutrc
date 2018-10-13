var apiPrefix = "api";

function login(pass, callback) {
    "use strict"
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

function submitTask(target, object, successCallback, failureCallback) {
    "use strict"
    var xhr = $.ajax({
        method: "POST",
        url: apiPrefix + "/tasks?" + jQuery.param({target: target}),
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

function getErrorMessage(resp) {
    var msg
    if (resp.responseJSON != undefined) {
        msg = resp.responseJSON.msg
    } else {
        msg = resp.statusText
    }
    return msg
}
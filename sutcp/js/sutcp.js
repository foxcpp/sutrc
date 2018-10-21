function groupPresentInDOM(id) {
    return $("#agents-group-" + id).length != 0
}

function addGroupToDOM(name, id) {
    $("#agentslist").append('\
    <div class="agents-group-root">\
        <a role="button" href="#" class="twoheader styleless-link agents-group-header" data-toggle="collapse" data-target="#agents-group-' + id + '">\
            <span class="twoheader-left h5">' + name + '</span>\
            <span id="agents-group-' + id + '-counters" class="twoheader-left h6 agents-group-counter">(...)</span>\
            <span class="twoheader-right fas fa-arrow-down fa-lg" />\
        </a>\
        <div class="collapse agents-group" data-parent="#agentslist" data-id="' + id + '" id="agents-group-' + id + '" aria-expanded="false">\
            <button type="button" data-role="broadcast-task" data-target="' + id + '" class="btn btn-floating broadcast-btn">\
                ${BROADCAST_TASK_BTN}\
            </button>\
        </div>\
        <hr>\
    </div>')
}

function addAgentToDOM(group, name, online) {
    var title = name
    var disabledAttr = ""
    var statusClass = ""
    if (online) {
        title += "${ONLINE_SUFFIX}"
        statusClass = "online-agent"
    } else {
        disabledAttr = "disabled"
        statusClass = "offline-agent"
    }
    
    $("#agents-group-" + group).append('\
                            <figure data-id="' + name + '" class="twoheader agent-entry ' + statusClass+ '">\
                                <div class="twoheader-left" datadata-target="' + name + '">\
                                    <span class="agent-name">' + title + '</span>\
                                    <button type="button" data-role="rename-agent" data-target="' + name + '" class="btn btn-transparent btn-dim agent-btn">\
                                        <span class="fas fa-sm fa-pencil-alt"></span>\
                                    </button>\
                                </div>\
                                <div class="twoheader-right">\
                                    <button type="button" ' + disabledAttr + ' data-role="browse-fs" data-target="' + name + '" class="btn btn-outline-secondary agent-btn">\
                                        ${BROWSE_FS_BTN}\
                                    </button>\
                                    <button type="button" ' + disabledAttr + ' data-role="send-task" data-target="' + name + '" class="btn btn-outline-secondary agent-btn">\
                                        ${SEND_TASK_BTN}\
                                    </button>\
                                </div>\
                            </figure>')
}

function removeEmptyGroups() {
    $(".agents-group:not(:has(.agent-entry))").parent(".agents-group-root").remove()
}

function updateGroupCounts() {
    var groups = $(".agents-group")
    for (var i = 0; i < groups.length; i++) {
        var id = groups[i].dataset.id
        
        var total = $("#agents-group-" + id).children(".agent-entry").length
        var online = total - $("#agents-group-" + id).children(".offline-agent").length
        
        $("#agents-group-" + id + "-counters").text(`${COUNTERS}`)
    }
}

function showAlertGeneric(id, type, where, text) {
    $("#" + id).alert("close")
    $(where).prepend('<div class="alert ' + type + ' alert-dismissible" id="' + id + '" role="alert">' + text + '.')
}

function showAlert(id, where, text) {
    showAlertGeneric(id, "alert-danger", where, text)
}

function showNotify(id, where, text) {
    showAlertGeneric(id, "alert-info", where, text)
}

function groupOnlineAgents(id) {
    var res = []
    var onlineAgents = $("#agents-group-" + id).children(".online-agent")
    for (var i = 0; i < onlineAgents.length; i++) {
        res.push(onlineAgents[i].dataset.id)
    }
    return res
}

function haveFSParent(path) {
    var parts = path.split("\\")
    return !(parts.length == 2 && parts[1] == "")
}

function parentFSPath(path) {
    if (path.endsWith("\\")) {
        return path.split("\\").slice(0, -2).join("\\") + "\\"
    } else {
        return path.split("\\").slice(0, -1).join("\\") + "\\"
    }
}

function filename(path) {
    if (path.endsWith("\\")) {
        return path.split("\\").slice(-2)
    } else {
        return path.split("\\").slice(-1)
    }
}

function addUpperDirEntry() {
    $("#fs-browser-body").append('\
        <div id="upper-dir-entry" class="fs-entry directory twoheader">\
            <span class="twoheader-left">\
                <a href="#" class="styleless-link" id="upper-dir-link">..</a>\
            </span>\
        </div>')
}

function addFSEntryToDOM(entry) {
    var dirClass = ""
    if (entry.dir) {
        dirClass = "directory"
    }
    
    $("#fs-browser-body").append('\
                        <div class="twoheader fs-entry ' + dirClass + '" data-path="' + escapeHTML(entry.fullpath) + '">\
                            <span class="twoheader-left">\
                                <a href="#" class="styleless-link fs-link">' + entry.name + '</a>\
                                <button type="button" class="fs-rename-btn btn btn-transparent btn-dim">\
                                    <span class="fas fa-sm fa-pencil-alt"></span>\
                                </button>\
                            </span>\
                            <span class="twoheader-right">\
                                <button type="button" class="fs-delete-btn btn btn-sm btn-outline-danger">\
                                    <span class="fas fa-trash-alt"></span>\
                                </button>\
                            </span>\
                        </div>')
}

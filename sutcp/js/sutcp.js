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
        <div class="collapse agents-group" data-parent="#agentslist" data-id="' + id + '" id="agents-group-' + id + '" aria-expanded="false"></div>\
        <hr>\
    </div>')
}

function addAgentToDOM(group, name, online) {
    var title = name
    var disabledAttr = ""
    var offlineClass = ""
    if (online) {
        title += " (online)"
    } else {
        disabledAttr = "disabled"
        offlineClass = "offline-agent"
    }
    
    $("#agents-group-" + group).append('\
                            <figure class="twoheader agent-entry ' + offlineClass+ '">\
                                <div class="twoheader-left">\
                                    <span class="agent-name">' + title + '</span>\
                                    <button type="button" data-role="rename-agent" data-target="' + name + '" class="btn btn-transparent btn-dim agent-btn">\
                                        <span class="fas fa-sm fa-pencil-alt"></span>\
                                    </button>\
                                </div>\
                                <div class="twoheader-right">\
                                    <button type="button" ' + disabledAttr + ' data-role="send-task" data-target="' + name + '" class="btn btn-outline-secondary agent-btn">\
                                        Send task\
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
        var id = groups[0].dataset.id
        
        var total = $("#agents-group-" + id).children(".agent-entry").length
        var online = total - $("#agents-group-" + id).children(".offline-agent").length
        
        $("#agents-group-" + id + "-counters").text("(" + String(online) + " online, " + String(total) + " total)")
    }
}

function showAlert(id, where, text) {
    $("#" + id).alert("close")
    $(where).prepend('<div class="alert alert-danger alert-dismissible" id="' + id + '" role="alert">Failed to update agents list: ' + text + '.')
}
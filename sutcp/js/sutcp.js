/*

** How groups are represented and manipulated in page DOM

Each group gets agent list container with id in form agent-group-XXX, where XXX is 
name of the group. For example, agents 210-4, 210-2, 210-foobar belong to group 210.
So they will be added to DOM element with id agent-group-210.

*/

// Check if group element is already added to DOM.
//
// name - group name, i.e. 210.
function groupPresentInDOM(name) {
    return $("#agents-group-" + name).length != 0
}

// Add new group element to page.
//
// name - element title (something like "Group #210").
// id   - internal group name, will be used to refer to group
//        in all other code.
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

// Agent list DOM generator utility. Add agent to list in DOM.
//
// group - internal name of group to which agent belongs.
// name  - agent name (it will be used both for display and for internal identification).
// online - agent availability status (true = online, false = offline).
//
// Agent entry added with online=false will have interaciton buttons (send task, browse FS) disabled.
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
                                    <button type="button" data-role="delete-agent" data-target="' + name + '" class="btn btn-transparent btn-dim small-btn agent-btn">\
                                        <span aria-label="Delete agent" class="fas fa-times"></span>\
                                    </button>\
                                    <button style="margin-left: 20px" type="button" data-role="rename-agent" data-target="' + name + '" class="btn btn-transparent btn-dim small-btn agent-btn">\
                                        <span aria-label="Rename agent" class="fas fa-sm fa-pencil-alt"></span>\
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

// Agent list DOM generator utility. Remove empty (without agents) groups from DOM.
//
// See populate loadAgentsList for details.
function removeEmptyGroups() {
    $(".agents-group:not(:has(.agent-entry))").parent(".agents-group-root").remove()
}

// Agent list DOM generator utility. Update agent list counters for each group.
//
// Agent list counters are these (M online, N total) in title of each group entry.
// This function will update values in DOM in corrodance with information about
// agents **already added to DOM**.
function updateGroupCounts() {
    var groups = $(".agents-group")
    for (var i = 0; i < groups.length; i++) {
        var id = groups[i].dataset.id
        
        var total = $("#agents-group-" + id).children(".agent-entry").length
        var online = total - $("#agents-group-" + id).children(".offline-agent").length
        
        $("#agents-group-" + id + "-counters").text(`${COUNTERS}`)
    }
}

// Show alert message somewhere on page.
//
// type - CSS class used for alert styling.
// You probably want to use one of Bootstrap's styles here:
// - alert-danger
// - alert-info
// - alert-warning
// etc, see Bootstrap docs.
// 
// id - unique alert ID.
// If alert with same id already exists on page - it will removed.
//
// where - CSS selector of alert parent element.
// Alert will be added before first $(where)'s children.
//
// text - Alert contents (text).
function showAlertGeneric(id, type, where, text) {
    $("#" + id).alert("close")
    $(where).prepend('<div class="alert ' + type + ' alert-dismissible" id="' + id + '" role="alert">' + text + '.')
}

// Wrapper for showAlertGeneric, shows alert with type set to alert-danger.
function showAlert(id, where, text) {
    showAlertGeneric(id, "alert-danger", where, text)
}

// Wrapper for showAlertGeneric, shows alert with type set to alert-info.
function showNotify(id, where, text) {
    showAlertGeneric(id, "alert-info", where, text)
}

// Returns list of IDs of online agents from certain group with specified name.
function groupOnlineAgents(id) {
    var res = []
    var onlineAgents = $("#agents-group-" + id).children(".online-agent")
    for (var i = 0; i < onlineAgents.length; i++) {
        res.push(onlineAgents[i].dataset.id)
    }
    return res
}

// Filesystem path utility. Check if passed path have some logical parent element.
//
// For example, C:\ do not have parent, but C:\Windows does (its parent is C:\).
function haveFSParent(path) {
    var parts = path.split("\\")
    return !(parts.length == 2 && parts[1] == "")
}

// Filesystem path utility. Return logical parent of passed path.
//
// Example: parentFSPath("C:\Windows") == "C:\".
// Invalid value will be returned is path doesn't have a logical parent.
function parentFSPath(path) {
    if (path.endsWith("\\")) {
        return path.split("\\").slice(0, -2).join("\\") + "\\"
    } else {
        return path.split("\\").slice(0, -1).join("\\") + "\\"
    }
}

// Filesystem path utility. Get last element from path.
//
// For example filename("C:\\foobar") will return "foobar".
function filename(path) {
    if (path.endsWith("\\")) {
        return path.split("\\").slice(-2)[0]
    } else {
        return path.split("\\").slice(-1)[0]
    }
}

// File browser DOM generator utility. Add ".." meta-directory
// to top of the list of current view.
//
// Added entry will get ID upper-dir-link.
function addUpperDirEntry() {
    $("#fs-browser-body").append('\
        <div id="upper-dir-entry" class="fs-entry directory twoheader">\
            <span class="twoheader-left">\
                <a href="#" class="styleless-link" id="upper-dir-link">..</a>\
            </span>\
        </div>')
}

// File browser DOM generator utility. Add filesystem entry returned by server
// to end of current view.
//
// entry is object as returned by agent in task result object.
// So it must have entry.dir, entry.name and entry.fullpath fields.
// Entry DOM root will get data-path attribute with value from entry.fullpath.
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

function prepareBroadcastContainer(carousel) {
    $("#single-result").hide()
    $("#screenshot-result").hide()
    $("#broadcast-result").children().remove()
    $("#broadcast-result").show()
    $("#broadcast-result").attr("carousel", carousel)

    // Insert carousel skeleton.
    if (carousel) {
        $("#broadcast-result").append('\
            <div id="result-carousel" class="carousel" data-interval="false">\
                <div class="carousel-inner">\
                </div>\
                <a class="carousel-control-prev" href="#result-carousel" role="button" data-slide="prev">\
                    <span class="fas fa-arrow-left" aria-hidden="true"></span>\
                    <span class="sr-only">Previous</span>\
                </a>\
                <a class="carousel-control-next" href="#result-carousel" role="button" data-slide="next">\
                    <span class="fas fa-arrow-right" aria-hidden="true"></span>\
                    <span class="sr-only">Next</span>\
                </a>\
            </div>')
        $("#result-container").attr("style", "padding: 0;")
        registerCarouselHandlers()
    } else {
        $("#result-container").removeAttr("style")
    }
}

function addBroadcastResult(label, contents) {
    var carousel = $("#broadcast-result").attr("carousel") == "true"
    if (carousel) {
        console.log("carousel attr")
        $("#result-carousel").children(".carousel-inner").append('<div data-label="' + label + '" class="carousel-item">' + contents + '</div>')

        var first = $(".carousel-item:first")
        if (!first.hasClass("active")) {
            first.addClass("active")
        }
    } else {
        $("#broadcast-result").append('<b>' + label + '</b><div>' + contents + '</div>')
    }
}

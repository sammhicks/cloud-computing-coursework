"use strict";

var eventSource;
var sessionToken;

function uploadFile(name, type, body) {
    return new Promise(function (resolve, reject) {
        const xmlHttp = new XMLHttpRequest();

        xmlHttp.onreadystatechange = function () {
            if (this.readyState == this.DONE) {
                if (this.status == 200) {
                    resolve(this);
                } else {
                    reject(this);
                }
            }
        }

        xmlHttp.open("POST", "/upload?" + $.param({
            "name": name,
            "token": sessionToken
        }, true));
        xmlHttp.setRequestHeader("Content-type", type);
        xmlHttp.send(body);
    })
}

function onGoogleSignIn(googleUser) {
    $("body").addClass("signed-in").addClass("loading-history");

    $("#user-name").text(googleUser.getBasicProfile().getName());

    const token = googleUser.getAuthResponse().id_token;

    eventSource = new EventSource("/events?token=" + token);

    eventSource.onerror = function (ev) {
        console.log("Event Source closed:", ev);
        this.close()
        $("body").addClass("disconnected");
    }

    eventSource.onmessage = function (ev) {
        if (sessionToken == undefined) {
            $("body").removeClass("loading-history");
            sessionToken = ev.data.trim();
        } else {
            const message = JSON.parse(atob(ev.data.trim()));

            switch (message.Type) {
                case "text/x-clipboard":
                    $("<div/>", {
                        "x-created": message.Created
                    }).append($("<textarea/>", {
                        "val": message.Body
                    })).appendTo("#received-items");
                    break;
                default:
                    $("<div/>", {
                        "x-created": message.Created
                    }).append($("<a/>", {
                        "text": message.Name,
                        "href": message.URL,
                        "target": "blank"
                    })).appendTo("#received-items");
                    break;
            }
        }
    }
}

function loadFile(file) {
    return new Promise(function (resolve, reject) {
        const fileReader = new FileReader();

        fileReader.onabort = function () {
            reject();
        }

        fileReader.onerror = function () {
            reject();
        }

        fileReader.onprogress = function (ev) {
            console.log(ev);
        }

        fileReader.onloadend = function (ev) {
            console.log(ev);
            resolve(this.result);
        };

        fileReader.readAsArrayBuffer(file);
    });
}

async function uploadFiles(files) {
    $("body").addClass("uploading");

    for (let i = 0; i < files.length; i++) {
        await uploadFile(files[i].name, files[i].type, await loadFile(files[i]));
    }

    $("body").removeClass("uploading");
}

$.when($.ready).then(function () {
    $("#sign-out").click(function () {
        var auth2 = gapi.auth2.getAuthInstance();

        auth2.disconnect();

        auth2.signOut().then(function () {
            console.log('User signed out.');

            $("body").removeClass("signed-in");
            if (eventSource) {
                eventSource.close();
                eventSource = undefined;
            }
            sessionToken = undefined;

            $("#user-name").text("");

            $("#paste-form textarea").val("");

            $("#received-items").empty();
        });

    })

    $("#paste-form").submit(async function (transmitEvent) {
        transmitEvent.preventDefault();

        await uploadFile("Clipboard", "text/x-clipboard", $(transmitEvent.target).find("textarea").val());
    });

    $("#file-drop-area").on("dragstart drag", function (ev) {
        ev.preventDefault();
    }).on("dragenter dragover", function (ev) {
        const types = ev.originalEvent.dataTransfer.types;

        if ($.inArray("Files", types) > -1) {
            $(this).addClass("valid-dragging").removeClass("invalid-dragging");

            ev.preventDefault();
        } else {
            $(this).addClass("invalid-dragging").removeClass("valid-dragging");
        }
    }).on("dragleave dragend drop", function (ev) {
        $(this).removeClass("valid-dragging").removeClass("invalid-dragging");

        ev.preventDefault();
    }).on("drop", async function (ev) {
        const types = ev.originalEvent.dataTransfer.types;

        if ($.inArray("Files", types) > -1) {
            const files = ev.originalEvent.dataTransfer.files;

            await uploadFiles(files);
        }

        ev.preventDefault();
    });
});

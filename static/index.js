"use strict";

var websocketLock;

class Lock {
    constructor(item) {
        this.promise = Promise.resolve();
        this.item = item;
    }

    lock(action) {
        this.promise = this.promise.then(() => action(this.item));

        return this.promise;
    }
}

function onGoogleSignIn(googleUser) {
    $("body").addClass("signedin");

    $("#username").text(googleUser.getBasicProfile().getName());

    const token = googleUser.getAuthResponse().id_token;

    const host = window.location.host;
    const protocol = window.location.protocol == "http:" ? "ws" : "wss";

    const websocket = new WebSocket(protocol + "://" + host + "/ws");

    websocket.onopen = function (ev) {
        websocket.send(token)
    }

    websocket.onclose = function (ev) {
        console.log("Websocket closed:", ev);
        $("body").addClass("disconnected");
    }

    websocket.onmessage = function (ev) {
        const message = JSON.parse(ev.data);

        console.log("Message:", message);

        switch (message.Type) {
            case "text/x-clipboard":
                $("<div/>", {
                    "x-created": message.Created
                }).append($("<textarea/>", {
                    "val": message.Body
                })).appendTo("#receiveditems");
                break;
            default:
                $("<div/>", {
                    "x-created": message.Created
                }).append($("<a/>", {
                    "text": message.Name,
                    "href": message.URL,
                    "target": "blank"
                })).appendTo("#receiveditems");
                break;
        }
    }

    websocket.onerror = console.error;

    websocketLock = new Lock(websocket);
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

        }

        fileReader.onloadend = function (ev) {
            resolve(this.result);
        };

        fileReader.readAsArrayBuffer(file);
    });
}

async function uploadFiles(files) {
    $("body").addClass("uploading");

    try {
        for (let i = 0; i < files.length; i++) {
            await websocketLock.lock(async function (websocket) {
                websocket.send(JSON.stringify({
                    "Name": files[i].name,
                    "Type": files[i].type
                }))
                const body = await loadFile(files[i]);

                websocket.send(body);
            });
        }
    } catch (e) {
        console.error(e)
    }

    $("body").removeClass("uploading");
}

$.when($.ready).then(function () {
    /*$("#signout").click(function () {
        var auth2 = gapi.auth2.getAuthInstance();
        auth2.signOut().then(function () {
            console.log('User signed out.');
            $("body").removeClass("signedin");
            if (websocketLock) {
                websocketLock.lock(function (websocket) {
                    websocket.close();
                })
            }
        });

    })*/

    $("#pasteform").submit(function (transmitEvent) {
        transmitEvent.preventDefault();

        if (websocketLock) {
            var message = $(transmitEvent.target).find("textarea").val();

            websocketLock.lock(function (websocket) {
                websocket.send(JSON.stringify({
                    "Type": "text/x-clipboard"
                }))

                websocket.send(message);
            });
        }
    });

    $("#filedroparea").on("dragstart drag", function (ev) {
        ev.preventDefault();
    }).on("dragenter dragover", function (ev) {
        const types = ev.originalEvent.dataTransfer.types;

        console.log(types);

        if ($.inArray("Files", types) > -1) {
            $(this).addClass("validDragging").removeClass("invalidDragging");

            ev.preventDefault();
        } else {
            $(this).addClass("invalidDragging").removeClass("validDragging");
        }
    }).on("dragleave dragend drop", function (ev) {
        $(this).removeClass("validDragging").removeClass("invalidDragging");

        ev.preventDefault();
    }).on("drop", function (ev) {
        const types = ev.originalEvent.dataTransfer.types;

        if ($.inArray("Files", types)) {
            const files = ev.originalEvent.dataTransfer.files;

            console.log("Dropped!");
            uploadFiles(files);
        }

        ev.preventDefault();
    });
});

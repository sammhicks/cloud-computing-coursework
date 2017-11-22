"use strict";

var websocket;

function onGoogleSignIn(googleUser) {
    $("body").addClass("signedin");

    var token = googleUser.getAuthResponse().id_token;

    var host = window.location.host;
    var protocol = window.location.protocol == "http:" ? "ws" : "wss";

    websocket = new WebSocket(protocol + "://" + host + "/ws");

    websocket.onopen = function (ev) {
        $("body").addClass("signedin");

        websocket.send(token)
    }

    websocket.onclose = function (ev) {
        console.log("Websocket closed:", ev);
    }

    websocket.onmessage = function (ev) {
        $("<a/>", {
            text: "Message",
            href: "https://storage.cloud.google.com/cloud-computing-coursework.appspot.com/" + ev.data,
            target: "blank"
        }).wrap("<li/>").parent().appendTo("#snippets")
    }

    websocket.onerror = console.error;
}

$.when($.ready).then(function () {
    $("#transmit").submit(function (transmitEvent) {
        transmitEvent.preventDefault();

        if (websocket) {
            var message = $(transmitEvent.target).find("[name=snippet]").val();

            websocket.send(message);
        }
    });
});

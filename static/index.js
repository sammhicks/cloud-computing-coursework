$.when($.ready).then(function () {
    var auth2; // The Sign-In object.
    var googleUser; // The current user.

    var websocket;

    var initSigninV2 = function () {
        auth2 = gapi.auth2.init({
            client_id: '812818444262-dihtcq1cl07rrc4d3gs86obfs95dhe4i.apps.googleusercontent.com',
            scope: 'profile'
        });

        auth2.attachClickHandler('g-signin2');

        auth2.isSignedIn.listen(signinChanged);

        auth2.currentUser.listen(userChanged);

        if (auth2.isSignedIn.get() == true) {
            auth2.signIn();
        }
    };


    var signinChanged = function (val) {
        console.log('Signin state changed to ', val);

        if (!val) {
            if (websocket) {
                websocket.close()
            }
        }
    };

    var userChanged = function (user) {
        console.log('User now: ', user);
        googleUser = user;

        if (websocket) {
            websocket.close()
        }

        var token = user.getAuthResponse().id_token;

        var host = window.location.host;
        var protocol = window.location.protocol == "http:" ? "ws" : "wss";

        websocket = new WebSocket(protocol + "://" + host + "/ws");

        websocket.onopen = function (ev) {
            $("body").addClass("signedin");

            websocket.send(token)
        }

        websocket.onclose = function (ev) {
            console.log("Websocket closed:", ev);

            $("body").removeClass("signedin");
        }

        websocket.onmessage = function (ev) {
            $("<a/>", {
                text: "Message",
                href: "https://storage.cloud.google.com/cloud-computing-coursework.appspot.com/" + ev.data,
                target: "blank"
            }).wrap("<li/>").parent().appendTo("#snippets")
        }

        websocket.onerror = console.error;
    };

    $("#transmit").submit(function (transmitEvent) {
        transmitEvent.preventDefault();

        if (auth2 && googleUser && websocket) {
            var message = $(transmitEvent.target).find("[name=snippet]").val();

            websocket.send(message);
        }
    });

    gapi.load('auth2', initSigninV2);
});

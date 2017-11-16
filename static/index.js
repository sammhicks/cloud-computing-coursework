$.when($.ready).then(function () {
    var auth2; // The Sign-In object.
    var googleUser; // The current user.

    var eventSource;

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

        if (val) {
            $("body").addClass("signedin");
        } else {
            $("body").removeClass("signedin");
        }
    };

    var userChanged = function (user) {
        console.log('User now: ', user);
        googleUser = user;

        if (eventSource) {
            eventSource.close();
        }

        var token = user.getAuthResponse().id_token;

        eventSource = new EventSource("/events?token=" + token);
        eventSource.onmessage = function (event) {
            $("<a/>", {
                text: "Message",
                href: "https://storage.cloud.google.com/cloud-computing-coursework.appspot.com/" + event.data,
                target: "blank"
            }).wrap("<li/>").parent().appendTo("#snippets")
        };
    };

    $("#transmit").submit(function (transmitEvent) {
        transmitEvent.preventDefault();

        if (auth2 && googleUser) {
            var token = googleUser.getAuthResponse().id_token
            var message = $(transmitEvent.target).find("[name=snippet]").val();

            $.post("/transmit?token=" + token, message, null, "text").done(function () {
                console.log("Transmitted ", message);
            }).fail(function (jqXHR, textStatus, errorThrown) {
                console.group("Transmission failed:")
                console.error("Transmit Event:", transmitEvent);
                console.error("Status:", textStatus);
                console.error("Error Thrown:", errorThrown);
                console.groupEnd();
            });
        }
    });

    gapi.load('auth2', initSigninV2);
});

$.when($.ready).then(function () {
    $("#transmit").submit(function (transmitEvent) {
        transmitEvent.preventDefault();

        var message = $(transmitEvent.target).find("[name=snippet]").val();
        $.post("/transmit", message, null, "text/plain").done(function () {
            console.log("Transmitted ", message);
        }).fail(function (jqXHR, textStatus, errorThrown) {
            console.group("Transmission failed:")
            console.error("Transmit Event:", transmitEvent);
            console.error("Status:", textStatus);
            console.error("Error Thrown:", errorThrown);
            console.groupEnd();
        });
    });

    var source = new EventSource("/events");
    source.onmessage = function (event) {
        console.log(atob(event.data))
    };
});

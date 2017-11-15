$.when($.ready).then(function () {
    $("#transmit").submit(function (transmitEvent) {
        transmitEvent.preventDefault();

        var target = $(transmitEvent.target);

        var message = target.find("[name=snippet]").val();
        $.post("/transmit", target.serialize()).done(function () {
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

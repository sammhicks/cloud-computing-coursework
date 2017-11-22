function uploadFile(file) {
    const progressBar = $("<progress/>", {
        "value": 0,
        "max": file.size
    });
    return new Promise(function (resolve, reject) {
        $("#progressbars").append(progressBar);

        const fileReader = new FileReader();

        fileReader.onabort = function () {
            reject();
        }

        fileReader.onerror = function () {
            reject();
        }

        fileReader.onprogress = function (ev) {
            progressBar.attr("value", ev.loaded);
        }

        fileReader.onloadend = function (ev) {
            console.log(this.result);

            resolve();
        };

        fileReader.readAsBinaryString(file);
    }).then(function (value) {
        progressBar.remove();
        return value;
    }, function (reason) {
        progressBar.remove();
        return reason;
    });
}

async function uploadFiles(files) {
    $("body").addClass("uploading");

    try {
        for (let i = 0; i < files.length; i++) {
            await uploadFile(files[i]);
        }
    } catch (e) { }

    $("body").removeClass("uploading");
}

$.when($.ready).then(function () {
    const allowedMimeTypes = $(["Files", "text/plain"]);

    $("#filedrop").on("dragstart drag", function (ev) {
        ev.preventDefault();
    }).on("dragenter dragover", function (ev) {
        const types = ev.originalEvent.dataTransfer.types;

        const allowedTypes = allowedMimeTypes.filter(types);

        if (allowedTypes.length > 0) {
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

        const allowedTypes = allowedMimeTypes.filter(types);

        if (allowedTypes.length > 0) {
            switch (allowedTypes[0]) {
                case "Files":
                    const files = ev.originalEvent.dataTransfer.files;

                    uploadFiles(files);
                    break;
                default:
                    console.log("Text:", ev.originalEvent.dataTransfer.getData(allowedTypes[0]));

                    break;
            }
        }

        ev.preventDefault();
    });
});

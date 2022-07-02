const artistURL = "http://localhost:8080/artist/";
const addBTN = document.getElementById("addBTN");
const removeBTN = document.getElementById("removeBTN");

addBTN.addEventListener("click", function(e) {
    let s = getSUI();
    if (s.length == 0) {
        window.alert("Please enter an artist's Spotify URI");
    } else {
        addArtist(s)
        .then(status => {
            console.log(status ? "ADDED" : "FAILED");
        });
    }
});

removeBTN.addEventListener("click", function(e) {
    let s = getSUI();
    if (s.length == 0) {
        window.alert("Please enter an artist's Spotify URI");
    } else {
        removeArtist(s)
        .then(status => {
            console.log(status ? "REMOVED" : "FAILED");
        });
    }
});

async function addArtist(sui) {
    const url = artistURL + "add?sui=" + sui;
    const res = await fetch(url);

    return res.status;
}

async function removeArtist(sui) {
    const url = artistURL + "remove?sui=" + sui;
    const res = await fetch(url);

    return res.status;
}

function getSUI() {
    return document.getElementById("SUI").value;
}
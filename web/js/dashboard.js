const artistURL = "http://localhost:8080/artist/";
const addBTN = document.getElementById("addBTN");
const removeBTNs = document.getElementsByClassName("removeBTN");

addBTN.addEventListener("click", function(e) {
    let s = document.getElementById("SUI").value;
    if (s.length == 0) {
        window.alert("Please enter an artist's Spotify URI");
    } else {
        addArtist(s)
        .then(status => {
            if (status != 200) {
                window.alert("Was not able to add artist");
            };
        });
    }
});

for (let i = 0; removeBTNs.length > i; i++) {
    removeBTNs[i].addEventListener("click", function(e) {
        let btn = e.target;
        let sui = btn.parentNode.dataset.sui;
        if (s.length == 0) {
            window.alert("Please enter an artist's Spotify URI");
        } else {
            removeArtist(sui)
            .then(status => {
                if (status != 200) {
                    window.alert("Was not able to remove artist");
                } else {
                    document.querySelector('[data-sui="'+sui+'"]').remove();
                    document.getElementById("totalArtists").innerHTML -= 1;
                };
            });
        }
    });
}

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
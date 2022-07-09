const apiURL = getApiURL();
const addBTN = document.getElementById("addBTN");
const removeBTNs = document.getElementsByClassName("removeBTN");
const approveBTNs = document.getElementsByClassName("approveBTN");

class Artist {
    constructor(name, SUI) {
        this.Name = name;
        this.SUI = SUI;
    }
}

addBTN.addEventListener("click", (e) => {
    let s = document.getElementById("SUI").value;
    if (s.length == 0) {
        window.alert("Please enter an artist's Spotify URI");
    } else {
        addArtist(s)
        .then(res => res.json())
        .then(data => {
            if (data.status != 200) {
                window.alert("Was not able to add artist");
            } else if (data.status == 200) {
                artistListAdd(new Artist(data.name, s));
            }
        });
    }
});

for (let i = 0; removeBTNs.length > i; i++) {
    removeBTNs[i].addEventListener("click", (e) => removeBTNListener(e));
}

function removeBTNListener(e) {
    let btn = e.target;
    let sui = btn.parentNode.dataset.sui;
    if (sui.length == 0) {
        window.alert("Please enter an artist's Spotify URI");
    } else {
        removeArtist(sui)
        .then(status => {
            if (status != 200) {
                window.alert("Was not able to remove artist");
            } else if (status == 200) {
                document.querySelector('[data-sui="'+sui+'"]').remove();
                document.getElementById("totalArtists").innerHTML -= 1;
            }
        });
    }
}

for (let i = 0; approveBTNs.length > i; i++) {
    approveBTNs[i].addEventListener("click", (e) => approveBTNsListener(e));
}

function approveBTNsListener(e) {
    let btn = e.target;
    let sui = btn.parentNode.dataset.sui;
    if (sui.length == 0) {
        window.alert("Please enter a track's Spotify URI");
    } else {
        approveTrack(sui)
        .then(status => {
            if (status != 200) {
                window.alert("Was not able to remove artist");
            } else if (status == 200) {
                document.querySelector('[data-sui="'+sui+'"]').remove();
                document.getElementById("totalArtists").innerHTML -= 1;
            }
        });
    }
}

function artistListAdd(artist) {
    let listEle = document.getElementById("artistList");
    let artistEle = document.createElement("li");
    artistEle.dataset.sui = artist.SUI;
    
    let a = document.createElement("a");
    classAdder(a, "six columns");
    a.href = `https://open.spotify.com/artist/${artist.SUI}`;
    a.innerHTML = artist.Name;

    let p = document.createElement("p");
    classAdder(p, "five columns");
    p.innerHTML = artist.SUI;

    let b = document.createElement("input");
    classAdder(b, "one column u-pull-right removeBTN");
    b.value = "X";
    b.type = "button";
    b.addEventListener("click", (e) => removeBTNListener(e));

    artistEle.appendChild(a);
    artistEle.appendChild(p);
    artistEle.appendChild(b);
    listEle.appendChild(artistEle);
}


// API Calls
async function addArtist(sui) {
    const url = apiURL + "artist/add?sui=" + sui;
    const res = await fetch(url);

    return res;
}

async function removeArtist(sui) {
    const url = apiURL + "artist/remove?sui=" + sui;
    const res = await fetch(url);

    return res.status;
}

async function approveTrack(sui) {
    const url = apiURL + "trackreview/reviewed?sui=" + sui + "?status=approved";
    const res = await fetch(url);

    return res.status;
}

// Helper Functions
function classAdder(cl, classes) {
    let c = classes.split(" ");
    c.forEach((c) => {
        cl.classList.add(c);
    });
}

function getApiURL() {
    return window.location.href.includes("localhost") ? "http://localhost:8080/api/" : "https://quiet-reaches-27997.herokuapp.com";
};
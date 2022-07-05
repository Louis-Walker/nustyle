const artistURL = "http://localhost:8080/artist/";
const addBTN = document.getElementById("addBTN");
const removeBTNs = document.getElementsByClassName("removeBTN");

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

// API Calls
async function addArtist(sui) {
    const url = artistURL + "add?sui=" + sui;
    const res = await fetch(url);

    return res;
}

async function removeArtist(sui) {
    const url = artistURL + "remove?sui=" + sui;
    const res = await fetch(url);

    return res.status;
}
// END

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

// Helper Functions
function classAdder(cl, classes) {
    let c = classes.split(" ");
    c.forEach((c) => {
        cl.classList.add(c);
    });
}
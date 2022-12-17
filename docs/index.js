import {
    create,
    search,
    insertBatch,
} from "https://unpkg.com/@lyrasearch/lyra@latest/dist/esm/src/lyra.js";

let restaurantDB;

async function createLyraInstance(event) {
    const endpoint = "data.json";
    const response = await fetch(endpoint);
    const data = await response.json();

    restaurantDB = create({
        schema: {
            Name: "string",
            Address: "string",
            Location: "string",
            Price: "string",
            Cuisine: "string",
            Longitude: "number",
            Latitude: "number",
            PhoneNumber: "number",
            Url: "string",
            WebsiteUrl: "string",
            Award: "string",
            FacilitiesAndServices: "string",
        },
    });
    await insertBatch(restaurantDB, data, { batchSize: 500 });
}

function jsonToHtmlTable(json) {
    const table = document.getElementById("search-results");

    // Clear existing table
    table.innerHTML = "";
    if (!json[0]) return;

    // Create table header
    const ignoreList = [
        "Longitude",
        "Latitude",
        "Url",
        "WebsiteUrl",
        "FacilitiesAndServices",
    ];
    const thead = table.createTHead();
    let row = thead.insertRow();
    let columns = Object.keys(json[0].document);
    for (let key of columns) {
        // Skip keys that should be ignored
        if (ignoreList.includes(key)) continue;

        let th = document.createElement("th");
        let text = document.createTextNode(key);
        th.appendChild(text);
        row.appendChild(th);
    }

    // Create table body
    const tbody = table.createTBody();
    for (let element of json) {
        row = tbody.insertRow();
        for (let key of columns) {
            // Skip keys that should be ignored
            if (ignoreList.includes(key)) continue;

            let cell = row.insertCell();

            // If the key is "Name", create a link element and set its href attribute
            if (key === "Name") {
                let link = document.createElement("a");
                link.setAttribute("href", element.document["Url"]);
                link.innerText = element.document[key];
                cell.appendChild(link);
            } else {
                let text = document.createTextNode(element.document[key]);
                cell.appendChild(text);
            }
        }
    }
}

function handleSearch(event) {
    const searchTerm = event.target.value;
    const searchResult = search(restaurantDB, {
        term: searchTerm,
        properties: ["Name", "Address", "Location", "Cuisine"],
        limit: 50,
        tolerance: 3,
    });
    jsonToHtmlTable(searchResult.hits);
}

window.addEventListener("load", createLyraInstance);
document.body.addEventListener("input", handleSearch);

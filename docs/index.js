import {
    create,
    search,
    insertBatch,
    formatNanoseconds,
} from "https://unpkg.com/@lyrasearch/lyra@latest/dist/esm/src/lyra.js";

let restaurantDB;
const table = document.getElementById("search-results");

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
            Longitude: "string",
            Latitude: "string",
            PhoneNumber: "string",
            Url: "string",
            WebsiteUrl: "string",
            Award: "string",
            FacilitiesAndServices: "string",
        },
    });
    await insertBatch(restaurantDB, data, { batchSize: 500 });
}

function jsonToHtmlTable(json) {
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

            // Add relevant attribute based on key.
            // E.g. if the key is "Name", create a link element and set its href attribute
            switch (key) {
                case "Name": {
                    const websiteUrl = element.document["WebsiteUrl"];
                    if (!websiteUrl) {
                        const text = document.createTextNode(
                            element.document[key],
                        );
                        cell.appendChild(text);
                        break;
                    }
                    const link = document.createElement("a");
                    link.setAttribute("href", websiteUrl);
                    link.innerText = element.document[key];
                    cell.appendChild(link);
                    break;
                }
                case "PhoneNumber": {
                    const phoneNumber = element.document["PhoneNumber"];
                    if (phoneNumber) {
                        let text = document.createTextNode(
                            element.document[key],
                        );
                        cell.appendChild(text);
                    }
                    break;
                }
                case "Award": {
                    const link = document.createElement("a");
                    const url = element.document["Url"];
                    link.setAttribute("href", url);
                    link.innerText = element.document[key];
                    cell.appendChild(link);
                    break;
                }
                default: {
                    const text = document.createTextNode(element.document[key]);
                    cell.appendChild(text);
                    break;
                }
            }
        }
    }
}

function handleSearch(event) {
    const searchTerm = document.getElementById("search-term").value;
    if (!searchTerm || !restaurantDB) {
        table.innerHTML = "";
        return;
    }

    const searchResult = search(restaurantDB, {
        term: searchTerm,
        properties: ["Name", "Address", "Location", "Cuisine"],
        limit: 50,
        tolerance: 3,
    });
    jsonToHtmlTable(searchResult.hits);
}

window.onload = async function (event) {
    await createLyraInstance(event);
    handleSearch(event);
};

document.body.addEventListener("input", handleSearch);

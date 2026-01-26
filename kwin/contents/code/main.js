const service = "io.github.muesli.DeckMaster";
const path = "/Monitor";
const method = "ActiveWindowChanged";

function activeWindowChanged(window) {
    if (window) {
        const name = window.resourceName + "." + window.resourceClass;
        const caption = window.caption;
        const id = window.internalId.toString();
        callDBus(service, path, service, method, name, caption, id);
    }
}

const signal = workspace.windowActivated ?? workspace.clientActivated
signal.connect(activeWindowChanged);

print("enabled: DeckMaster");

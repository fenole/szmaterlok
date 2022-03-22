const eventStreamResource = "/stream";
const sseTypes = ["message-sent"];

function setupSSE() {
  let evtSource = new EventSource(eventStreamResource, {
    withCredentials: true,
  });

  const handleEvent = (event) => {
    document.dispatchEvent(
      new CustomEvent("sse:" + event.type, {
        bubbles: true,
        detail: {
          data: JSON.parse(event.data),
        },
      }),
    );
  };

  sseTypes.forEach((eventType) => {
    evtSource.addEventListener(eventType, handleEvent);
  });

  return evtSource;
}

function sendMessage(msg) {
  fetch("/message", {
    method: "POST",
    credentials: "include",
    body: JSON.stringify({ content: msg }),
  });
}

document.addEventListener("alpine:init", () => {
  setupSSE();

  Alpine.data("messages", () => ({
    messages: [],

    formatDate(sentAt) {
      return new Date(sentAt).toLocaleTimeString("en-gb", {});
    },
  }));

  Alpine.data("messageInput", () => ({
    newMessage: "",

    send() {
      sendMessage(this.newMessage);
      this.newMessage = "";
    },
  }));
});

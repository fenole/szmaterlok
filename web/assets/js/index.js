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

document.addEventListener("alpine:init", () => {
  Alpine.data("eventCounter", () => ({
    count: 0,
    message: "",
    lastEventSendAt: 0,

    handleMessage(event) {
      this.message = event.detail.data.msg;
      this.lastEventSendAt = event.detail.data.sendAt;
      this.count++;
    },
  }));

  setupSSE();
});

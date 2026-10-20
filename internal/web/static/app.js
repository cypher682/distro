const lineRows = document.querySelector("#order-lines");
const lineTemplate = document.querySelector("#line-template");
const addLine = document.querySelector("[data-add-line]");

if (lineRows && lineTemplate && addLine) {
  addLine.addEventListener("click", () => {
    lineRows.append(lineTemplate.content.cloneNode(true));
  });

  lineRows.addEventListener("click", (event) => {
    const button = event.target.closest("[data-remove-line]");
    if (!button || lineRows.children.length === 1) {
      return;
    }
    button.closest(".line-row").remove();
  });
}

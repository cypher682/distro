document.querySelectorAll("[data-line-builder]").forEach((builder) => {
  const lineRows = builder.querySelector("[data-line-rows]");
  const lineTemplate = builder.querySelector("template[data-line-template]");
  const addLine = builder.querySelector("[data-add-line]");

  if (!lineRows || !lineTemplate || !addLine) {
    return;
  }

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
});

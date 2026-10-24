// Bedienkomfort für die Anmelde- und Registrieren-Felder in der Kopfleiste.
// Ohne JavaScript bleiben die details-Elemente normal bedienbar, der Hero-Link
// springt dann nur zum Anker #registrieren.

const registerPanel = document.getElementById("registrieren");

function openRegister() {
  if (!registerPanel) {
    return;
  }
  registerPanel.open = true;
  const firstField = registerPanel.querySelector("input");
  if (firstField) {
    firstField.focus();
  }
}

for (const link of document.querySelectorAll("[data-open-register]")) {
  link.addEventListener("click", openRegister);
}

// Direktaufruf mit /#registrieren öffnet das Feld ebenfalls.
if (window.location.hash === "#registrieren") {
  openRegister();
}

// Klick außerhalb schließt ein offenes Feld, Escape ebenso.
document.addEventListener("click", (event) => {
  for (const panel of document.querySelectorAll(".auth-panel[open]")) {
    if (!panel.contains(event.target)) {
      panel.open = false;
    }
  }
});

document.addEventListener("keydown", (event) => {
  if (event.key !== "Escape") {
    return;
  }
  for (const panel of document.querySelectorAll(".auth-panel[open]")) {
    panel.open = false;
    const summary = panel.querySelector("summary");
    if (summary) {
      summary.focus();
    }
  }
});

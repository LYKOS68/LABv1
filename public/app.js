const form = document.querySelector("#auth-form");
const message = document.querySelector("#message");
let mode = "register";

document.querySelectorAll("[data-mode]").forEach((button) => button.addEventListener("click", () => {
  mode = button.dataset.mode;
  document.querySelectorAll("[data-mode]").forEach((item) => item.classList.toggle("active", item === button));
  document.querySelector(".enter").innerHTML = `${mode === "register" ? "Registrieren" : "Einloggen"} <span>→</span>`;
  document.querySelector("#password").autocomplete = mode === "register" ? "new-password" : "current-password";
  message.textContent = "";
}));

form.addEventListener("submit", async (event) => {
  event.preventDefault();
  message.textContent = "Türsteher prüft …";
  const response = await fetch(`/api/${mode}`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ username: username.value, password: password.value }) });
  const result = await response.json();
  if (!response.ok) return message.textContent = result.error;
  localStorage.setItem("rap-lab-token", result.token);
  document.querySelector("#artist").textContent = result.user.username;
  document.querySelector("#token").textContent = result.token;
  form.closest(".gate").hidden = true;
  document.querySelector("#studio").hidden = false;
});

document.querySelector("#copy").addEventListener("click", async () => {
  await navigator.clipboard.writeText(document.querySelector("#token").textContent);
  message.textContent = "Token kopiert.";
});

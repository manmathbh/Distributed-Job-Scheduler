/* Public client registration UI. Loaded after app.js so it can reuse the dashboard helpers. */
(function () {
  function registrationBody() {
    return '<div class="form-grid">' +
      '<div class="field full"><label>Name</label><input class="input" name="name" required minlength="2" maxlength="80" placeholder="Your name" autocomplete="name" /></div>' +
      '<div class="field full"><label>Email</label><input class="input" name="email" type="email" required maxlength="254" placeholder="you@example.com" autocomplete="email" /></div>' +
      '<div class="field full"><div class="muted">Your account uses an API key as its scheduler credential. No password is required.</div></div>' +
      '</div>';
  }

  function openRegistrationDialog() {
    openDialog(dialogShell('Create account', registrationBody()));
    const form = $('#dlgForm');
    if (form) form.addEventListener('submit', submitRegistration);
  }

  async function submitRegistration(event) {
    event.preventDefault();
    const form = $('#dlgForm');
    if (!form) return;
    const fd = new FormData(form);
    const name = String(fd.get('name') || '').trim();
    const email = String(fd.get('email') || '').trim();

    try {
      const res = await fetch('/api/v1/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, email })
      });

      let body = {};
      try { body = await res.json(); } catch (e) {}
      if (!res.ok) throw new Error(body.error || res.statusText || 'Registration failed');

      api.key = body.api_key;
      localStorage.setItem('scheduler_api_key', api.key);
      const keyInput = $('#apiKey');
      if (keyInput) keyInput.value = api.key;
      state.projects = [];
      state.currentProject = null;

      openDialog(
        '<div class="dialog"><div class="dialog-head"><div class="dialog-title">Account created</div><button class="icon-btn" onclick="closeDialog()">' + icon('x') + '</button></div>' +
        '<div class="dialog-body">' +
        '<div class="card" style="margin-bottom:14px"><div class="card-title" style="margin-bottom:6px">Welcome, ' + esc(body.name) + '</div><div class="muted">Your client account is ready. Your API key has also been saved in this browser.</div></div>' +
        '<div class="field"><label>API key — save this somewhere safe</label><input class="input mono" value="' + esc(body.api_key) + '" readonly id="newApiKey" /></div>' +
        '<div class="muted" style="margin-top:8px">This key is your scheduler credential. It is returned only during registration.</div>' +
        '</div>' +
        '<div class="dialog-foot"><button type="button" class="btn btn-secondary" id="copyApiKeyBtn">Copy key</button><button type="button" class="btn" id="continueDashboardBtn">Continue</button></div></div>'
      );

      const copyBtn = $('#copyApiKeyBtn');
      if (copyBtn) copyBtn.addEventListener('click', async () => {
        try {
          await navigator.clipboard.writeText(body.api_key);
          toast('API key copied', 'ok');
        } catch (e) {
          toast('Copy failed — select the key and copy it manually', 'error');
        }
      });

      const continueBtn = $('#continueDashboardBtn');
      if (continueBtn) continueBtn.addEventListener('click', () => {
        closeDialog();
        location.hash = '#/queues';
        render();
      });
    } catch (e) {
      toast(e.message, 'error');
    }
  }

  function initRegistration() {
    const button = $('#registerBtn');
    if (button) button.addEventListener('click', openRegistrationDialog);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initRegistration);
  } else {
    initRegistration();
  }
})();

// Email confirmation script — loaded externally to satisfy CSP
(function () {
    var scriptEl = document.currentScript;
    var token = scriptEl ? scriptEl.getAttribute('data-token') : '';

    if (!token) {
        showError('Токен подтверждения не найден. Проверьте письмо и попробуйте снова.');
        return;
    }

    fetch('/api/v1/auth/confirm', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: token })
    })
        .then(function (r) {
            return r.json().then(function (d) {
                return { ok: r.ok, data: d };
            });
        })
        .then(function (res) {
            if (res.ok) {
                showSuccess('Email успешно подтверждён! Теперь вы можете войти в систему.');
            } else {
                var msg = typeof res.data === 'string'
                    ? res.data
                    : (res.data ? JSON.stringify(res.data) : 'Ошибка подтверждения');
                showError(msg);
            }
        })
        .catch(function () {
            showError('Произошла ошибка соединения. Попробите позже.');
        });

    function showSuccess(msg) {
        hideSpinner();
        var el = document.getElementById('result');
        el.textContent = msg;
        el.className = 'result success';
        el.style.display = 'block';
        document.getElementById('backLink').style.display = 'block';
    }

    function showError(msg) {
        hideSpinner();
        var el = document.getElementById('result');
        el.textContent = msg;
        el.className = 'result error';
        el.style.display = 'block';
        document.getElementById('backLink').style.display = 'block';
    }

    function hideSpinner() {
        var spinner = document.getElementById('spinner');
        if (spinner) spinner.style.display = 'none';
        var waitText = document.querySelector('.container > p');
        if (waitText) waitText.style.display = 'none';
    }
})();

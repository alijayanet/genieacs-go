/**
 * apiFetch — wrapper de fetch que injeta automaticamente o token JWT.
 * Redireciona para / se não houver token ou se receber 401.
 */
async function apiFetch(url, options = {}) {
    const token = localStorage.getItem('token');
    const headers = Object.assign({}, options.headers || {});
    if (token) {
        headers['Authorization'] = 'Bearer ' + token;
    }
    const res = await fetch(url, Object.assign({}, options, { headers }));
    if (res.status === 401) {
        localStorage.removeItem('token');
        window.location.href = '/';
        return null;
    }
    return res;
}

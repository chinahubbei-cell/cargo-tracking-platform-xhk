const http = require('http');

const data = JSON.stringify({
  username: 'admin',
  password: 'admin123'
});

const req = http.request(
  'http://localhost:8001/api/admin/auth/login',
  { method: 'POST', headers: { 'Content-Type': 'application/json' } },
  res => {
    let body = '';
    res.on('data', chunk => body += chunk);
    res.on('end', () => console.log('Status:', res.statusCode, 'Body:', body));
  }
);
req.on('error', console.error);
req.write(data);
req.end();

"""
Simple HTTP server for previewing GO-ACS Web UI
Run with: python preview_server.py
"""
import http.server
import socketserver
import os
import json
from urllib.parse import urlparse, parse_qs

PORT = 8080
os.chdir(os.path.dirname(os.path.abspath(__file__)))

class GOACSHandler(http.server.SimpleHTTPRequestHandler):
    def do_GET(self):
        parsed = urlparse(self.path)
        path = parsed.path
        
        # Route pages
        if path == '/' or path == '/index.html':
            self.serve_file('web/templates/index.html')
        elif path == '/dashboard':
            self.serve_file('web/templates/dashboard.html')
        elif path == '/devices':
            self.serve_file('web/templates/devices.html')
        elif path.startswith('/device/'):
            self.serve_file('web/templates/device-detail.html')
        elif path == '/provisions':
            self.serve_file('web/templates/provisions.html')
        elif path == '/map':
            self.serve_file('web/templates/map.html')
        elif path == '/customers':
            self.serve_file('web/templates/customers.html')
        elif path == '/billing':
            self.serve_file('web/templates/billing.html')
        elif path == '/portal' or path == '/portal/':
            self.serve_file('web/templates/portal.html')
        elif path == '/portal/login':
            self.serve_file('web/templates/portal-login.html')
        elif path == '/packages':
            self.serve_file('web/templates/packages.html')
        elif path == '/tasks':
            self.serve_file('web/templates/tasks.html')
        elif path == '/logs':
            self.serve_file('web/templates/logs.html')
        elif path == '/settings':
            self.serve_file('web/templates/settings.html')
        
        # Static files
        elif path.startswith('/static/'):
            file_path = 'web' + path
            if os.path.exists(file_path):
                self.serve_file(file_path)
            else:
                self.send_error(404)
        
        # Mock API
        elif path == '/api/dashboard/stats':
            self.send_json({
                "totalDevices": 5,
                "onlineDevices": 3,
                "offlineDevices": 2,
                "pendingTasks": 1,
                "devicesByModel": {
                    "HG8245H": 2,
                    "F660": 2,
                    "AN5506": 1
                }
            })
        
        elif path == '/api/devices':
            self.send_json({
                "devices": [
                    {"id": 1, "serialNumber": "HWTC12345678", "manufacturer": "Huawei", "modelName": "HG8245H", "status": "online", "ipAddress": "192.168.1.1", "lastContact": "2026-01-04T22:50:00Z", "latitude": -6.2088, "longitude": 106.8456, "address": "Jl. Sudirman No. 1, Jakarta"},
                    {"id": 2, "serialNumber": "ZTEGC87654321", "manufacturer": "ZTE", "modelName": "F660", "status": "online", "ipAddress": "192.168.1.2", "lastContact": "2026-01-04T22:45:00Z", "latitude": -6.1751, "longitude": 106.8650, "address": "Jl. Thamrin No. 10, Jakarta"},
                    {"id": 3, "serialNumber": "FHTT11223344", "manufacturer": "FiberHome", "modelName": "AN5506-04-F", "status": "offline", "ipAddress": "192.168.1.3", "lastContact": "2026-01-04T20:00:00Z", "latitude": -6.2297, "longitude": 106.8295, "address": "Jl. Gatot Subroto No. 5, Jakarta"},
                    {"id": 4, "serialNumber": "HWTC99887766", "manufacturer": "Huawei", "modelName": "HG8245H5", "status": "online", "ipAddress": "192.168.1.4", "lastContact": "2026-01-04T22:52:00Z", "latitude": -6.1862, "longitude": 106.8225, "address": "Jl. Kebon Sirih No. 15, Jakarta"},
                    {"id": 5, "serialNumber": "ZTEGC55443322", "manufacturer": "ZTE", "modelName": "F670L", "status": "offline", "ipAddress": "192.168.1.5", "lastContact": "2026-01-04T18:00:00Z", "latitude": -6.2615, "longitude": 106.8106, "address": "Jl. TB Simatupang No. 22, Jakarta"},
                ],
                "total": 5
            })
        
        elif path.startswith('/api/devices/') and path.endswith('/wifi'):
            self.send_json({
                "ssid": "MyWiFi_Network",
                "password": "password123",
                "ssid5g": "MyWiFi_5G",
                "password5g": "password123",
                "channel": 6,
                "enabled": True
            })
        
        elif path.startswith('/api/devices/') and path.endswith('/wan'):
            self.send_json([
                {"id": 1, "name": "wan1_pppoe", "connectionType": "PPPoE", "status": "Connected", "vlan": 100},
                {"id": 2, "name": "wan2_bridge", "connectionType": "Bridge", "status": "Connected", "vlan": 200}
            ])
        
        elif path.startswith('/api/devices/') and path.endswith('/parameters'):
            self.send_json([
                {"path": "InternetGatewayDevice.DeviceInfo.UpTime", "value": "86400"},
                {"path": "InternetGatewayDevice.DeviceInfo.SoftwareVersion", "value": "V3R019C10S120"},
                {"path": "InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID", "value": "MyWiFi_Network"},
            ])
        
        elif path.startswith('/api/devices/') and path.endswith('/pon'):
            self.send_json({
                "rxPower": -18.5,
                "txPower": 2.3,
                "onuId": 1,
                "distance": 2.5,
                "temperature": 45,
                "voltage": 3.3,
                "biasCurrent": 15
            })
        
        elif path.startswith('/api/devices/') and path.endswith('/clients'):
            self.send_json({
                "clients": [
                    {"name": "iPhone 15 Pro", "mac": "A4:B2:C1:D3:E5:F7", "ip": "192.168.1.101", "type": "phone", "rssi": -45},
                    {"name": "MacBook Pro", "mac": "B8:C3:D2:E4:F6:A1", "ip": "192.168.1.102", "type": "laptop", "rssi": -52},
                    {"name": "Samsung Galaxy Tab", "mac": "C6:D4:E3:F5:A2:B8", "ip": "192.168.1.103", "type": "tablet", "rssi": -61},
                    {"name": "Smart TV", "mac": "D7:E5:F4:A6:B3:C9", "ip": "192.168.1.104", "type": "tv", "rssi": -38},
                    {"name": "Unknown Device", "mac": "E8:F6:A5:B7:C4:D2", "ip": "192.168.1.105", "type": "other", "rssi": -72}
                ]
            })
        
        elif path.startswith('/api/devices/'):
            device_id = path.split('/')[-1]
            self.send_json({
                "id": int(device_id) if device_id.isdigit() else 1,
                "serialNumber": "HWTC12345678",
                "manufacturer": "Huawei",
                "modelName": "HG8245H",
                "hardwareVersion": "V300R019",
                "softwareVersion": "V3R019C10S120",
                "status": "online",
                "ipAddress": "192.168.1.1",
                "lastContact": "2026-01-04T22:50:00Z"
            })
        
        # Billing API - Packages
        elif path == '/api/packages':
            self.send_json([
                {"id": 1, "name": "Home Basic", "description": "Basic internet", "downloadSpeed": 10, "uploadSpeed": 3, "price": 150000, "isActive": True},
                {"id": 2, "name": "Home Standard", "description": "Standard internet", "downloadSpeed": 20, "uploadSpeed": 5, "price": 250000, "isActive": True},
                {"id": 3, "name": "Home Premium", "description": "Premium internet", "downloadSpeed": 50, "uploadSpeed": 10, "price": 400000, "isActive": True},
                {"id": 4, "name": "Business", "description": "Business internet", "downloadSpeed": 100, "uploadSpeed": 20, "price": 750000, "isActive": True}
            ])
        
        # Billing API - Customers
        elif path == '/api/customers':
            self.send_json({
                "customers": [
                    {"id": 1, "customerCode": "CUST-0001", "name": "John Doe", "email": "john@email.com", "phone": "08123456789", "status": "active", "packageId": 2, "balance": 0},
                    {"id": 2, "customerCode": "CUST-0002", "name": "Ahmad Susanto", "email": "ahmad@email.com", "phone": "08234567890", "status": "active", "packageId": 3, "balance": -250000},
                    {"id": 3, "customerCode": "CUST-0003", "name": "Budi Wijaya", "email": "budi@email.com", "phone": "08345678901", "status": "suspended", "packageId": 1, "balance": -500000}
                ],
                "total": 3
            })
        
        # Billing API - Invoices
        elif path == '/api/invoices':
            self.send_json({
                "invoices": [
                    {"id": 1, "invoiceNo": "INV-202601-0001", "customerId": 1, "total": 250000, "status": "paid", "paidAmount": 250000, "dueDate": "2026-01-15"},
                    {"id": 2, "invoiceNo": "INV-202601-0002", "customerId": 2, "total": 400000, "status": "pending", "paidAmount": 0, "dueDate": "2026-01-15"},
                    {"id": 3, "invoiceNo": "INV-202601-0003", "customerId": 3, "total": 150000, "status": "overdue", "paidAmount": 0, "dueDate": "2026-01-01"}
                ],
                "total": 3
            })
        
        # Billing API - Payments
        elif path == '/api/payments':
            self.send_json({
                "payments": [
                    {"id": 1, "paymentNo": "PAY-202601-0001", "customerId": 1, "amount": 250000, "paymentMethod": "transfer", "status": "completed", "paymentDate": "2026-01-03"},
                    {"id": 2, "paymentNo": "PAY-202512-0015", "customerId": 2, "amount": 400000, "paymentMethod": "cash", "status": "completed", "paymentDate": "2026-01-02"}
                ],
                "total": 2
            })
        
        # Billing Stats
        elif path == '/api/billing/stats':
            self.send_json({
                "totalCustomers": 142,
                "activeCustomers": 134,
                "suspendedCustomers": 8,
                "monthlyRevenue": 35500000,
                "pendingInvoices": 23,
                "overdueAmount": 2500000,
                "todayPayments": 1750000
            })
        
        else:
            super().do_GET()
    
    def do_POST(self):
        path = urlparse(self.path).path
        content_length = int(self.headers.get('Content-Length', 0))
        body = self.rfile.read(content_length) if content_length > 0 else b''
        
        if path == '/api/auth/login':
            try:
                data = json.loads(body)
                if data.get('username') == 'admin' and data.get('password') == 'admin123':
                    self.send_json({
                        "success": True,
                        "token": "mock-jwt-token-12345",
                        "user": {"username": "admin", "role": "admin"}
                    })
                else:
                    self.send_json({"success": False, "error": "Invalid credentials"}, 401)
            except:
                self.send_json({"success": False, "error": "Invalid request"}, 400)
        
        # Customer Portal Login
        elif path == '/api/portal/auth/login':
            try:
                data = json.loads(body)
                # Mock customer credentials: johndoe / password123 or CUST-0001 / password123
                valid_users = {
                    'johndoe': {'id': 1, 'code': 'CUST-0001', 'name': 'John Doe', 'password': 'password123'},
                    'CUST-0001': {'id': 1, 'code': 'CUST-0001', 'name': 'John Doe', 'password': 'password123'},
                    'ahmad': {'id': 2, 'code': 'CUST-0002', 'name': 'Ahmad Susanto', 'password': 'password123'},
                    'CUST-0002': {'id': 2, 'code': 'CUST-0002', 'name': 'Ahmad Susanto', 'password': 'password123'}
                }
                username = data.get('username', '')
                password = data.get('password', '')
                
                if username in valid_users and valid_users[username]['password'] == password:
                    user = valid_users[username]
                    self.send_json({
                        "success": True,
                        "token": f"customer-{user['id']}-mock-token",
                        "customer": {
                            "id": user['id'],
                            "customerCode": user['code'],
                            "name": user['name'],
                            "email": f"{username.lower()}@email.com",
                            "status": "active"
                        }
                    })
                else:
                    self.send_json({"success": False, "error": "Invalid username or password"}, 401)
            except:
                self.send_json({"success": False, "error": "Invalid request"}, 400)
        
        elif path == '/api/auth/logout' or path == '/api/portal/auth/logout':
            self.send_json({"success": True})
        
        elif '/reboot' in path or '/refresh' in path:
            self.send_json({"success": True, "message": "Command queued"})
        
        elif path == '/api/devices':
            self.send_json({"success": True, "id": 6})
        
        else:
            self.send_json({"success": True})
    
    def do_PUT(self):
        self.send_json({"success": True, "message": "Updated"})
    
    def serve_file(self, filepath):
        try:
            with open(filepath, 'rb') as f:
                content = f.read()
            
            self.send_response(200)
            if filepath.endswith('.html'):
                self.send_header('Content-Type', 'text/html; charset=utf-8')
            elif filepath.endswith('.css'):
                self.send_header('Content-Type', 'text/css')
            elif filepath.endswith('.js'):
                self.send_header('Content-Type', 'application/javascript')
            else:
                self.send_header('Content-Type', 'application/octet-stream')
            self.send_header('Content-Length', len(content))
            self.end_headers()
            self.wfile.write(content)
        except FileNotFoundError:
            self.send_error(404, f'File not found: {filepath}')
    
    def send_json(self, data, status=200):
        content = json.dumps(data).encode('utf-8')
        self.send_response(status)
        self.send_header('Content-Type', 'application/json')
        self.send_header('Content-Length', len(content))
        self.send_header('Access-Control-Allow-Origin', '*')
        self.end_headers()
        self.wfile.write(content)
    
    def log_message(self, format, *args):
        print(f"[{self.log_date_time_string()}] {args[0]}")

if __name__ == '__main__':
    with socketserver.TCPServer(("", PORT), GOACSHandler) as httpd:
        print(f"""
================================================================
                    GO-ACS Preview Server
================================================================
  Web UI:     http://localhost:{PORT}
  Dashboard:  http://localhost:{PORT}/dashboard
  Devices:    http://localhost:{PORT}/devices
  Packages:   http://localhost:{PORT}/packages
  Customers:  http://localhost:{PORT}/customers
  Billing:    http://localhost:{PORT}/billing
  Map:        http://localhost:{PORT}/map
================================================================
  Login:      admin / admin123
  Press Ctrl+C to stop
================================================================
        """)
        try:
            httpd.serve_forever()
        except KeyboardInterrupt:
            print("\nServer stopped.")


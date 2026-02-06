import 'package:flutter/material.dart';
import '../services/api_service.dart';
import 'home_screen.dart';

class LoginScreen extends StatefulWidget {
  @override
  _LoginScreenState createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  final _usernameController = TextEditingController();
  final _passwordController = TextEditingController();
  bool _isLoading = false;
  String _errorMessage = '';

  Future<void> _login() async {
    setState(() {
      _isLoading = true;
      _errorMessage = '';
    });

    final success = await ApiService().login(
      _usernameController.text,
      _passwordController.text,
    );

    setState(() {
      _isLoading = false;
    });

    if (success) {
      Navigator.pushReplacement(
        context,
        MaterialPageRoute(builder: (context) => HomeScreen()),
      );
    } else {
      setState(() {
        _errorMessage = 'Login failed. Check your credentials.';
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.white,
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(24.0),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Icon(Icons.wifi_tethering, size: 80, color: Colors.blue[800]),
              SizedBox(height: 16),
              Text(
                "GO-ACS",
                textAlign: TextAlign.center,
                style: TextStyle(fontSize: 32, fontWeight: FontWeight.bold, color: Colors.blue[800]),
              ),
              SizedBox(height: 8),
              Text(
                "Customer Portal Login",
                textAlign: TextAlign.center,
                style: TextStyle(fontSize: 16, color: Colors.grey[600]),
              ),
              SizedBox(height: 48),
              TextField(
                controller: _usernameController,
                decoration: InputDecoration(
                  labelText: "Username or Customer Code",
                  border: OutlineInputBorder(borderRadius: BorderRadius.circular(8)),
                  prefixIcon: Icon(Icons.person),
                ),
              ),
              SizedBox(height: 16),
              TextField(
                controller: _passwordController,
                obscureText: true,
                decoration: InputDecoration(
                  labelText: "Password",
                  border: OutlineInputBorder(borderRadius: BorderRadius.circular(8)),
                  prefixIcon: Icon(Icons.lock),
                ),
              ),
              SizedBox(height: 16),
              if (_errorMessage.isNotEmpty)
                Text(
                  _errorMessage,
                  style: TextStyle(color: Colors.red),
                  textAlign: TextAlign.center,
                ),
              SizedBox(height: 24),
              ElevatedButton(
                onPressed: _isLoading ? null : _login,
                child: Padding(
                  padding: const EdgeInsets.symmetric(vertical: 16.0),
                  child: _isLoading 
                    ? SizedBox(height: 20, width: 20, child: CircularProgressIndicator(color: Colors.white, strokeWidth: 2)) 
                    : Text("Login", style: TextStyle(fontSize: 16)),
                ),
                style: ElevatedButton.styleFrom(
                  backgroundColor: Colors.blue[800],
                  shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

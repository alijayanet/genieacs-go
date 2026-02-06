import 'dart:convert';
import 'package:http/http.dart' as http;
import 'package:shared_preferences/shared_preferences.dart';
import '../models/models.dart';

class ApiService {
  // Use 10.0.2.2 for Android Emulator to access localhost:8080
  // Change this to your server IP for physical devices
  static const String baseUrl = "http://10.0.2.2:8080/api";
  
  static final ApiService _instance = ApiService._internal();
  factory ApiService() => _instance;
  ApiService._internal();

  Customer? _currentUser;
  Customer? get currentUser => _currentUser;

  Future<bool> login(String username, String password) async {
    try {
      final response = await http.post(
        Uri.parse("$baseUrl/portal/auth/login"),
        headers: {"Content-Type": "application/json"},
        body: jsonEncode({"username": username, "password": password}),
      );

      if (response.statusCode == 200) {
        final data = jsonDecode(response.body);
        if (data['success'] == true) {
          _currentUser = Customer.fromJson(data['customer']);
          final prefs = await SharedPreferences.getInstance();
          await prefs.setString('user_data', jsonEncode(data['customer']));
          return true;
        }
      }
      return false;
    } catch (e) {
      print("Login error: $e");
      return false;
    }
  }

  Future<void> loadSession() async {
    final prefs = await SharedPreferences.getInstance();
    final userData = prefs.getString('user_data');
    if (userData != null) {
      _currentUser = Customer.fromJson(jsonDecode(userData));
    }
  }

  Future<void> logout() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove('user_data');
    _currentUser = null;
  }
  
  Future<List<BandwidthUsage>> getUsage() async {
    if (_currentUser == null) return [];
    try {
      final response = await http.get(Uri.parse("$baseUrl/mobile/usage?customerId=${_currentUser!.id}"));
      if (response.statusCode == 200) {
        final data = jsonDecode(response.body);
        if (data['success'] == true) {
           return (data['data'] as List).map((e) => BandwidthUsage.fromJson(e)).toList();
        }
      }
      return [];
    } catch (e) {
      return [];
    }
  }

  Future<List<Invoice>> getInvoices() async {
    if (_currentUser == null) return [];
    try {
      final response = await http.get(Uri.parse("$baseUrl/invoices?customerId=${_currentUser!.id}&limit=100"));
      if (response.statusCode == 200) {
        final data = jsonDecode(response.body);
        List<dynamic> list = data['invoices'];
        return list.map((e) => Invoice.fromJson(e)).toList();
      }
      return [];
    } catch (e) {
      return [];
    }
  }
}

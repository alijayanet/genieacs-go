import 'package:flutter/material.dart';
import 'screens/login_screen.dart';
import 'screens/home_screen.dart';
import 'services/api_service.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();
  
  // Load session
  await ApiService().loadSession();
  
  runApp(GoAcsApp());
}

class GoAcsApp extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    // Determine start screen
    final isLoggedIn = ApiService().currentUser != null;

    return MaterialApp(
      title: 'GO-ACS Mobile',
      theme: ThemeData(
        primarySwatch: Colors.blue,
        visualDensity: VisualDensity.adaptivePlatformDensity,
        scaffoldBackgroundColor: Colors.grey[50],
        appBarTheme: AppBarTheme(
          elevation: 0,
          backgroundColor: Colors.blue[800],
          foregroundColor: Colors.white,
        ),
      ),
      home: isLoggedIn ? HomeScreen() : LoginScreen(),
      debugShowCheckedModeBanner: false,
    );
  }
}

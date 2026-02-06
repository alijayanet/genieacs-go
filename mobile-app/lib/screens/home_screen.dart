import 'package:flutter/material.dart';
import 'package:fl_chart/fl_chart.dart';
import '../services/api_service.dart';
import '../models/models.dart';
import 'login_screen.dart';
import 'invoice_screen.dart';

class HomeScreen extends StatefulWidget {
  @override
  _HomeScreenState createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  int _currentIndex = 0;
  
  Customer? _customer;
  List<BandwidthUsage> _usage = [];
  bool _isLoading = true;

  @override
  void initState() {
    super.initState();
    _loadData();
  }

  Future<void> _loadData() async {
    _customer = ApiService().currentUser;
    // In real app, re-fetch profile to get updated balance
    final usage = await ApiService().getUsage();
    setState(() {
      _usage = usage;
      _isLoading = false;
    });
  }

  Future<void> _logout() async {
    await ApiService().logout();
    Navigator.pushAndRemoveUntil(
      context, 
      MaterialPageRoute(builder: (c) => LoginScreen()),
      (route) => false
    );
  }

  @override
  Widget build(BuildContext context) {
    // If lost session state in memory (e.g. reload), try to recover or show login
    if (_customer == null && !_isLoading) {
       return LoginScreen();
    }
    
    // Screens for bottom nav
    final screens = [
        _buildDashboard(),
        InvoiceScreen(),
        _buildProfile(),
    ];

    return Scaffold(
      backgroundColor: Colors.grey[100],
      appBar: AppBar(
        title: Text("GO-ACS"),
        backgroundColor: Colors.blue[800],
        foregroundColor: Colors.white,
        elevation: 0,
      ),
      body: _isLoading 
        ? Center(child: CircularProgressIndicator()) 
        : screens[_currentIndex],
      bottomNavigationBar: BottomNavigationBar(
        currentIndex: _currentIndex,
        onTap: (idx) => setState(() => _currentIndex = idx),
        selectedItemColor: Colors.blue[800],
        items: [
            BottomNavigationBarItem(icon: Icon(Icons.dashboard), label: "Dashboard"),
            BottomNavigationBarItem(icon: Icon(Icons.receipt_long), label: "Invoices"),
            BottomNavigationBarItem(icon: Icon(Icons.person), label: "Profile"),
        ],
      ),
    );
  }

  Widget _buildDashboard() {
    return SingleChildScrollView(
        padding: EdgeInsets.all(16),
        child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
                Text("Welcome back,", style: TextStyle(color: Colors.grey[600], fontSize: 16)),
                Text(_customer?.name ?? "User", style: TextStyle(fontSize: 24, fontWeight: FontWeight.bold, color: Colors.blue[900])),
                SizedBox(height: 24),
                
                // Bill Card
                _buildBillCard(),
                SizedBox(height: 24),
                
                // Usage Graph
                Text("Bandwidth Usage (Last 4 Hours)", style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold, color: Colors.blue[900])),
                SizedBox(height: 16),
                Container(
                    height: 220,
                    padding: EdgeInsets.only(right: 16, top: 16, bottom: 8),
                    decoration: BoxDecoration(
                        color: Colors.white,
                        borderRadius: BorderRadius.circular(16),
                        boxShadow: [BoxShadow(color: Colors.black.withOpacity(0.05), blurRadius: 10)]
                    ),
                    child: _usage.isEmpty 
                        ? Center(child: Text("No usage data available", style: TextStyle(color: Colors.grey)))
                        : LineChart(
                            LineChartData(
                                gridData: FlGridData(show: false),
                                titlesData: FlTitlesData(
                                  leftTitles: AxisTitles(sideTitles: SideTitles(showTitles: true, reservedSize: 40, getTitlesWidget: (val, media) {
                                    return Text("${val.toInt()} MB", style: TextStyle(fontSize: 10, color: Colors.grey));
                                  })),
                                  bottomTitles: AxisTitles(sideTitles: SideTitles(showTitles: false)),
                                  topTitles: AxisTitles(sideTitles: SideTitles(showTitles: false)),
                                  rightTitles: AxisTitles(sideTitles: SideTitles(showTitles: false)),
                                ),
                                borderData: FlBorderData(show: false),
                                lineBarsData: [
                                    // Download
                                    LineChartBarData(
                                        spots: _usage.asMap().entries.map((e) {
                                            // Convert bytes to MB
                                            return FlSpot(e.key.toDouble(), e.value.bytesReceived / 1024 / 1024);
                                        }).toList().reversed.toList(), // Reverse to show timeline correctly? DB returns desc.
                                        isCurved: true,
                                        color: Colors.green,
                                        barWidth: 3,
                                        dotData: FlDotData(show: false),
                                        belowBarData: BarAreaData(show: true, color: Colors.green.withOpacity(0.1)),
                                    ),
                                    // Upload
                                    LineChartBarData(
                                        spots: _usage.asMap().entries.map((e) {
                                            return FlSpot(e.key.toDouble(), e.value.bytesSent / 1024 / 1024);
                                        }).toList().reversed.toList(),
                                        isCurved: true,
                                        color: Colors.blue,
                                        barWidth: 3,
                                        dotData: FlDotData(show: false),
                                    )
                                ]
                            )
                        ),
                ),
                SizedBox(height: 8),
                Row(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    Container(width: 12, height: 12, color: Colors.green), SizedBox(width: 4), Text("Download"),
                    SizedBox(width: 16),
                    Container(width: 12, height: 12, color: Colors.blue), SizedBox(width: 4), Text("Upload"),
                  ],
                )
            ],
        ),
    );
  }
  
  Widget _buildBillCard() {
      // Simple logic: if balance < 0, it means credit? usually balance > 0 means debt in simple billing
      // Assuming balance is Amount Due.
      return Container(
          width: double.infinity,
          padding: EdgeInsets.all(24),
          decoration: BoxDecoration(
              gradient: LinearGradient(
                colors: [Colors.blue[800]!, Colors.blue[600]!],
                begin: Alignment.topLeft,
                end: Alignment.bottomRight,
              ),
              borderRadius: BorderRadius.circular(16),
              boxShadow: [BoxShadow(color: Colors.blue.withOpacity(0.4), blurRadius: 10, offset: Offset(0, 4))]
          ),
          child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                  Text("Total Amount Due", style: TextStyle(color: Colors.white70, fontSize: 14)),
                  SizedBox(height: 8),
                  Text(
                      "Rp ${_customer?.balance.toInt() ?? 0}", 
                      style: TextStyle(color: Colors.white, fontSize: 32, fontWeight: FontWeight.bold)
                  ),
                  SizedBox(height: 20),
                  SizedBox(
                    width: double.infinity,
                    child: ElevatedButton(
                        onPressed: () => setState(() => _currentIndex = 1), // Go to Invoices
                        child: Text("PAY NOW"),
                        style: ElevatedButton.styleFrom(
                            backgroundColor: Colors.white,
                            foregroundColor: Colors.blue[800],
                            elevation: 0,
                            padding: EdgeInsets.symmetric(vertical: 12),
                            shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8))
                        ),
                    ),
                  )
              ],
          ),
      );
  }



  Widget _buildProfile() {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          CircleAvatar(radius: 40, backgroundColor: Colors.blue[100], child: Icon(Icons.person, size: 40, color: Colors.blue)),
          SizedBox(height: 16),
          Text(_customer?.name ?? "", style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold)),
          Text(_customer?.email ?? "", style: TextStyle(color: Colors.grey)),
          SizedBox(height: 32),
          ElevatedButton.icon(
            onPressed: _logout, 
            icon: Icon(Icons.logout), 
            label: Text("Logout"),
            style: ElevatedButton.styleFrom(backgroundColor: Colors.red),
          )
        ],
      ),
    );
  }
}

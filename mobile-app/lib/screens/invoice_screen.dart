import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';
import '../models/models.dart';
import '../services/api_service.dart';

class InvoiceScreen extends StatefulWidget {
  @override
  _InvoiceScreenState createState() => _InvoiceScreenState();
}

class _InvoiceScreenState extends State<InvoiceScreen> {
  List<Invoice> _invoices = [];
  bool _isLoading = true;

  @override
  void initState() {
    super.initState();
    _loadInvoices();
  }

  Future<void> _loadInvoices() async {
    final invoices = await ApiService().getInvoices();
    setState(() {
      _invoices = invoices;
      _isLoading = false;
    });
  }
  
  Future<void> _payInvoice(Invoice invoice) async {
     // TODO: Implement Payment API Call
     // For now, redirect to user portal which is easier
     final url = Uri.parse("http://10.0.2.2:8080/portal");
     if (await canLaunchUrl(url)) {
       await launchUrl(url, mode: LaunchMode.externalApplication);
     } else {
       ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text("Could not launch payment page")));
     }
  }

  @override
  Widget build(BuildContext context) {
      if (_isLoading) return Center(child: CircularProgressIndicator());
      
      if (_invoices.isEmpty) return Center(child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(Icons.receipt_long, size: 64, color: Colors.grey[300]),
          SizedBox(height: 16),
          Text("No invoices found", style: TextStyle(color: Colors.grey)),
        ],
      ));

      return ListView.builder(
          padding: EdgeInsets.all(16),
          itemCount: _invoices.length,
          itemBuilder: (context, index) {
              final invoice = _invoices[index];
              return Card(
                  elevation: 2,
                  margin: EdgeInsets.only(bottom: 16),
                  shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                  child: Padding(
                      padding: EdgeInsets.all(16),
                      child: Column(
                          children: [
                              Row(
                                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                                  children: [
                                      Text(invoice.invoiceNo, style: TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
                                      Container(
                                          padding: EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                                          decoration: BoxDecoration(
                                              color: invoice.status == 'paid' ? Colors.green[100] : Colors.orange[100],
                                              borderRadius: BorderRadius.circular(12)
                                          ),
                                          child: Text(
                                              invoice.status.toUpperCase(),
                                              style: TextStyle(
                                                  color: invoice.status == 'paid' ? Colors.green[800] : Colors.orange[800],
                                                  fontSize: 12,
                                                  fontWeight: FontWeight.bold
                                              )
                                          ),
                                      )
                                  ],
                              ),
                              Padding(
                                padding: const EdgeInsets.symmetric(vertical: 12.0),
                                child: Divider(),
                              ),
                              Row(
                                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                                  children: [
                                      Text("Due Date", style: TextStyle(color: Colors.grey600)),
                                      Text(invoice.dueDate, style: TextStyle(fontWeight: FontWeight.w500)) 
                                  ],
                              ),
                              SizedBox(height: 8),
                              Row(
                                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                                  children: [
                                      Text("Total", style: TextStyle(color: Colors.grey600)),
                                      Text("Rp ${invoice.total.toStringAsFixed(0)}", style: TextStyle(fontWeight: FontWeight.bold, fontSize: 18, color: Colors.blue[900]))
                                  ],
                              ),
                              if (invoice.status != 'paid') ...[
                                  SizedBox(height: 16),
                                  SizedBox(
                                      width: double.infinity,
                                      child: ElevatedButton(
                                          onPressed: () => _payInvoice(invoice),
                                          child: Text("PAY ONLINE"),
                                          style: ElevatedButton.styleFrom(
                                              backgroundColor: Colors.blue[800],
                                              foregroundColor: Colors.white, 
                                              padding: EdgeInsets.symmetric(vertical: 12),
                                              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8))
                                          ),
                                      ),
                                  )
                              ]
                          ],
                      ),
                  ),
              );
          },
      );
  }
}

class Customer {
  final int id;
  final String customerCode;
  final String name;
  final String email;
  final String phone;
  final String address;
  final String status;
  final double balance;
  
  Customer({
    required this.id,
    required this.customerCode,
    required this.name,
    required this.email,
    required this.phone,
    required this.address,
    required this.status,
    required this.balance,
  });

  factory Customer.fromJson(Map<String, dynamic> json) {
    return Customer(
      id: json['id'] ?? 0,
      customerCode: json['customerCode'] ?? '',
      name: json['name'] ?? '',
      email: json['email'] ?? '',
      phone: json['phone'] ?? '',
      address: json['address'] ?? '',
      status: json['status'] ?? 'inactive',
      balance: (json['balance'] ?? 0).toDouble(),
    );
  }
}

class Invoice {
  final int id;
  final String invoiceNo;
  final String status;
  final double total;
  final String dueDate;
  final String createdAt;
  final List<InvoiceItem> items;

  Invoice({
    required this.id,
    required this.invoiceNo,
    required this.status,
    required this.total,
    required this.dueDate,
    required this.createdAt,
    this.items = const [],
  });

  factory Invoice.fromJson(Map<String, dynamic> json) {
    var itemsList = json['items'] as List? ?? [];
    List<InvoiceItem> items = itemsList.map((i) => InvoiceItem.fromJson(i)).toList();

    return Invoice(
      id: json['id'] ?? 0,
      invoiceNo: json['invoiceNo'] ?? '',
      status: json['status'] ?? 'pending',
      total: (json['total'] ?? 0).toDouble(),
      dueDate: json['dueDate'] ?? '',
      createdAt: json['createdAt'] ?? '',
      items: items,
    );
  }
}

class InvoiceItem {
  final String description;
  final double amount;

  InvoiceItem({required this.description, required this.amount});

  factory InvoiceItem.fromJson(Map<String, dynamic> json) {
    return InvoiceItem(
      description: json['description'] ?? '',
      amount: (json['amount'] ?? 0).toDouble(),
    );
  }
}

class BandwidthUsage {
  final String timestamp;
  final int bytesSent;
  final int bytesReceived;

  BandwidthUsage({
    required this.timestamp,
    required this.bytesSent,
    required this.bytesReceived,
  });

  factory BandwidthUsage.fromJson(Map<String, dynamic> json) {
    return BandwidthUsage(
      timestamp: json['timestamp'] ?? '',
      bytesSent: json['bytesSent'] ?? 0,
      bytesReceived: json['bytesReceived'] ?? 0,
    );
  }
}

class Ticket {
  final int id;
  final String ticketNo;
  final String subject;
  final String status;
  final String createdAt;

  Ticket({
    required this.id,
    required this.ticketNo,
    required this.subject,
    required this.status,
    required this.createdAt,
  });

  factory Ticket.fromJson(Map<String, dynamic> json) {
    return Ticket(
      id: json['id'] ?? 0,
      ticketNo: json['ticketNo'] ?? '',
      subject: json['subject'] ?? '',
      status: json['status'] ?? 'open',
      createdAt: json['createdAt'] ?? '',
    );
  }
}

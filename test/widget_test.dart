import 'package:flutter_test/flutter_test.dart';
import 'package:seyra/app/app.dart';
import 'package:seyra/core/constants/app_constants.dart';

void main() {
  testWidgets('Seyra application shell loads', (WidgetTester tester) async {
    await tester.pumpWidget(const SeyraApp());

    expect(find.text(AppConstants.appName), findsOneWidget);
  });
}

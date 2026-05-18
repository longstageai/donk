import 'dart:io';

import 'package:flutter/services.dart' show rootBundle;
import 'package:path_provider/path_provider.dart';

class ImgUtil {
  static Future<String> prepareTrayIconFromAssets(
    String assetPath, {
    String? fileName,
  }) async {
    final bytes = await rootBundle.load(assetPath);
    final dir = await getTemporaryDirectory();
    final out = File('${dir.path}/${fileName ?? 'app.png'}');
    await out.writeAsBytes(bytes.buffer.asUint8List());
    return out.path;
  }
}

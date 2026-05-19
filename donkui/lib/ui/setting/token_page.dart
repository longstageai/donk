import 'package:donk/app/layout/app_dialog.dart';
import 'package:flutter/material.dart';
import '../../common/service/token_service.dart';

/// Token用量统计页面
/// 显示用户的对话统计数据和Token使用详情
class TokenPage extends StatefulWidget {
  const TokenPage({super.key});

  @override
  State<TokenPage> createState() => _TokenPageState();
}

class _TokenPageState extends State<TokenPage> {
  /// 统计数据
  Map<String, dynamic> _statistics = {
    'todayConsumedTokens': 0,
    'todayRemainingTokens': 0,
    'usagePercent': 0.0,
    'isLimited': false,
  };

  /// Token使用详情列表
  List<Map<String, dynamic>> _tokenDetails = [];

  /// 是否正在加载
  bool _isLoading = true;

  /// 错误信息
  String? _errorMessage;

  /// 当前页码
  int _currentPage = 1;

  /// 每页条数
  final int _pageSize = 20;

  /// 是否还有更多数据
  bool _hasMoreData = true;

  /// 是否正在加载更多
  bool _isLoadingMore = false;

  @override
  void initState() {
    super.initState();
    _loadData();
  }

  /// 加载数据
  Future<void> _loadData() async {
    setState(() {
      _isLoading = true;
      _errorMessage = null;
    });

    try {
      // 并行加载预算状态和Token使用记录
      final results = await Future.wait([
        TokenService.getTokenBudget(),
        TokenService.getTokenUsage(page: _currentPage, pageSize: _pageSize),
      ]);

      final budgetResult = results[0];
      final usageResult = results[1];

      // 处理预算状态
      if (budgetResult['code'] == 0) {
        final budgetData = budgetResult['data'] as Map<String, dynamic>;
        if (!mounted) return;
        setState(() {
          _statistics = {
            'todayConsumedTokens': budgetData['used'] ?? 0,
            'todayRemainingTokens': budgetData['remaining'] ?? 0,
            'usagePercent': budgetData['usage_percent'] ?? 0.0,
            'isLimited': budgetData['is_limited'] ?? false,
          };
        });
      }

      // 处理Token使用记录
      if (usageResult['code'] == 0) {
        final usageData = usageResult['data'] as Map<String, dynamic>;
        final items = usageData['items'] as List<dynamic>? ?? [];
        final total = usageData['total'] as int? ?? 0;

        if (!mounted) return;
        setState(() {
          _tokenDetails =
              items.map((item) {
                final date = item['date'] as String? ?? '';
                final totalTokens = item['total_tokens'] as int? ?? 0;
                final updatedAt = item['updated_at'] as String? ?? '';

                // 格式化日期 20260422 -> 2026-04-22
                final formattedDate =
                    date.length == 8
                        ? '${date.substring(0, 4)}-${date.substring(4, 6)}-${date.substring(6, 8)}'
                        : date;

                // 格式化更新时间（转换为本地时区）
                String formattedUpdateTime = updatedAt;
                if (updatedAt.isNotEmpty && updatedAt.contains('T')) {
                  try {
                    // 解析 ISO8601 时间字符串
                    final utcDateTime = DateTime.parse(updatedAt);
                    // 转换为本地时区
                    final localDateTime = utcDateTime.toLocal();
                    // 格式化为本地时间字符串
                    formattedUpdateTime =
                        '${localDateTime.year}-${localDateTime.month.toString().padLeft(2, '0')}-${localDateTime.day.toString().padLeft(2, '0')} '
                        '${localDateTime.hour.toString().padLeft(2, '0')}:${localDateTime.minute.toString().padLeft(2, '0')}:${localDateTime.second.toString().padLeft(2, '0')}';
                  } catch (e) {
                    // 解析失败，使用原始字符串
                    formattedUpdateTime = updatedAt;
                  }
                }

                return {
                  'date': formattedDate,
                  'tokens': totalTokens,
                  'updatedAt': formattedUpdateTime,
                };
              }).toList();

          _hasMoreData = _tokenDetails.length < total;
        });
      }
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _errorMessage = '加载失败: $e';
      });
    } finally {
      if (mounted) {
        setState(() {
          _isLoading = false;
        });
      }
    }
  }

  /// 加载更多数据
  Future<void> _loadMoreData() async {
    if (_isLoadingMore || !_hasMoreData) return;

    setState(() {
      _isLoadingMore = true;
    });

    try {
      final nextPage = _currentPage + 1;
      final result = await TokenService.getTokenUsage(
        page: nextPage,
        pageSize: _pageSize,
      );

      if (result['code'] == 0) {
        final data = result['data'] as Map<String, dynamic>;
        final items = data['items'] as List<dynamic>? ?? [];
        final total = data['total'] as int? ?? 0;

        final newItems =
            items.map((item) {
              final date = item['date'] as String? ?? '';
              final totalTokens = item['total_tokens'] as int? ?? 0;
              final updatedAt = item['updated_at'] as String? ?? '';

              final formattedDate =
                  date.length == 8
                      ? '${date.substring(0, 4)}-${date.substring(4, 6)}-${date.substring(6, 8)}'
                      : date;

              // 格式化更新时间（转换为本地时区）
              String formattedUpdateTime = updatedAt;
              if (updatedAt.isNotEmpty && updatedAt.contains('T')) {
                try {
                  final utcDateTime = DateTime.parse(updatedAt);
                  final localDateTime = utcDateTime.toLocal();
                  formattedUpdateTime =
                      '${localDateTime.year}-${localDateTime.month.toString().padLeft(2, '0')}-${localDateTime.day.toString().padLeft(2, '0')} '
                      '${localDateTime.hour.toString().padLeft(2, '0')}:${localDateTime.minute.toString().padLeft(2, '0')}:${localDateTime.second.toString().padLeft(2, '0')}';
                } catch (e) {
                  formattedUpdateTime = updatedAt;
                }
              }

              return {
                'date': formattedDate,
                'tokens': totalTokens,
                'updatedAt': formattedUpdateTime,
              };
            }).toList();

        if (!mounted) return;
        setState(() {
          _tokenDetails.addAll(newItems);
          _currentPage = nextPage;
          _hasMoreData = _tokenDetails.length < total;
        });
      }
    } catch (e) {
      // 加载更多失败，不显示错误，保持现有数据
    } finally {
      if (mounted) {
        setState(() {
          _isLoadingMore = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          /// 页面标题
          _buildHeader(),
          const SizedBox(height: 8),

          /// 说明文字
          _buildDescription(),
          const SizedBox(height: 20),

          /// 统计数据卡片
          _buildStatisticsCard(),
          const SizedBox(height: 24),

          /// Token使用详情标题
          _buildDetailTitle(),
          const SizedBox(height: 16),

          /// Token使用详情表格
          Expanded(
            child:
                _isLoading
                    ? const Center(child: CircularProgressIndicator())
                    : _errorMessage != null
                    ? _buildErrorWidget()
                    : _buildTokenDetailsTable(),
          ),
        ],
      ),
    );
  }

  /// 构建错误提示
  Widget _buildErrorWidget() {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Text(_errorMessage!, style: const TextStyle(color: Colors.red)),
          const SizedBox(height: 16),
          ElevatedButton(onPressed: _loadData, child: const Text('重新加载')),
        ],
      ),
    );
  }

  /// 构建页面标题
  Widget _buildHeader() {
    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceBetween,
      children: [
        const Text(
          '用量统计',
          style: TextStyle(
            fontSize: 18,
            fontWeight: FontWeight.bold,
            color: Colors.black87,
          ),
        ),

        /// 关闭按钮
        MouseRegion(
          cursor: SystemMouseCursors.click,
          child: GestureDetector(
            onTap: () => AppDialog.dismiss(),
            child: const Icon(Icons.close, size: 20, color: Colors.grey),
          ),
        ),
      ],
    );
  }

  /// 构建说明文字
  Widget _buildDescription() {
    return const Text(
      '仅统计默认大模型的用量数据；不包含自定义模型数据',
      style: TextStyle(fontSize: 12, color: Colors.grey),
    );
  }

  /// 构建统计数据卡片
  Widget _buildStatisticsCard() {
    final isLimited = _statistics['isLimited'] as bool;
    final usagePercent = (_statistics['usagePercent'] as num).toDouble();

    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: const Color(0xFFF8F8F8),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceAround,
        children: [
          /// 今日消耗Token
          _buildStatItem(
            value: _formatNumber(_statistics['todayConsumedTokens'] as int),
            label: '今日消耗Token',
          ),

          /// 今日剩余Token
          _buildStatItem(
            value: _formatNumber(_statistics['todayRemainingTokens'] as int),
            label: '今日剩余Token',
            showInfoIcon: true,
          ),

          /// 剩余百分比
          if (isLimited)
            _buildStatItem(
              value: '${(100 - usagePercent).toStringAsFixed(1)}%',
              label: '剩余百分比',
            )
          else
            _buildStatItem(value: '无限制', label: '剩余百分比'),
        ],
      ),
    );
  }

  /// 构建单个统计项
  Widget _buildStatItem({
    required String value,
    required String label,
    bool showInfoIcon = false,
  }) {
    return Column(
      children: [
        Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(
              value,
              style: const TextStyle(
                fontSize: 24,
                fontWeight: FontWeight.bold,
                color: Colors.black87,
              ),
            ),
            if (showInfoIcon) ...[
              const SizedBox(width: 4),
              const Icon(Icons.info_outline, size: 14, color: Colors.grey),
            ],
          ],
        ),
        const SizedBox(height: 4),
        Text(label, style: const TextStyle(fontSize: 12, color: Colors.grey)),
      ],
    );
  }

  /// 构建Token使用详情标题
  Widget _buildDetailTitle() {
    return const Text(
      'Token使用详情',
      style: TextStyle(
        fontSize: 16,
        fontWeight: FontWeight.bold,
        color: Colors.black87,
      ),
    );
  }

  /// 构建Token使用详情表格
  Widget _buildTokenDetailsTable() {
    if (_tokenDetails.isEmpty) {
      return Container(
        decoration: BoxDecoration(
          color: const Color(0xFFF8F8F8),
          borderRadius: BorderRadius.circular(8),
        ),
        child: const Center(
          child: Text(
            '暂无数据',
            style: TextStyle(fontSize: 14, color: Colors.grey),
          ),
        ),
      );
    }

    return NotificationListener<ScrollNotification>(
      onNotification: (ScrollNotification scrollInfo) {
        if (scrollInfo.metrics.pixels >=
            scrollInfo.metrics.maxScrollExtent - 50) {
          _loadMoreData();
        }
        return false;
      },
      child: Container(
        decoration: BoxDecoration(
          color: const Color(0xFFF8F8F8),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Column(
          children: [
            /// 表头
            _buildTableHeader(),

            /// 分隔线
            const Divider(height: 1, color: Color(0xFFEEEEEE)),

            /// 表格内容
            Expanded(
              child: ListView.builder(
                itemCount: _tokenDetails.length,
                itemBuilder: (context, index) {
                  final item = _tokenDetails[index];
                  return _buildTableRow(
                    date: item['date'] as String,
                    tokens: item['tokens'] as int,
                    updatedAt: item['updatedAt'] as String,
                    isLast: index == _tokenDetails.length - 1,
                  );
                },
              ),
            ),

            /// 加载更多提示
            if (_isLoadingMore)
              Container(
                padding: const EdgeInsets.symmetric(vertical: 12),
                alignment: Alignment.center,
                child: const SizedBox(
                  width: 20,
                  height: 20,
                  child: CircularProgressIndicator(strokeWidth: 2),
                ),
              ),
          ],
        ),
      ),
    );
  }

  /// 构建表格表头
  Widget _buildTableHeader() {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      child: const Row(
        children: [
          Expanded(
            flex: 2,
            child: Text(
              '日期',
              style: TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.w500,
                color: Colors.black87,
              ),
            ),
          ),
          Expanded(
            flex: 2,
            child: Text(
              'Token消耗',
              style: TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.w500,
                color: Colors.black87,
              ),
            ),
          ),
          Expanded(
            flex: 3,
            child: Text(
              '更新时间',
              style: TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.w500,
                color: Colors.black87,
              ),
            ),
          ),
        ],
      ),
    );
  }

  /// 构建表格行
  Widget _buildTableRow({
    required String date,
    required int tokens,
    required String updatedAt,
    required bool isLast,
  }) {
    return Column(
      children: [
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
          child: Row(
            children: [
              Expanded(
                flex: 2,
                child: Text(
                  date,
                  style: const TextStyle(fontSize: 13, color: Colors.black87),
                ),
              ),
              Expanded(
                flex: 2,
                child: Text(
                  _formatNumber(tokens),
                  style: const TextStyle(fontSize: 13, color: Colors.black87),
                ),
              ),
              Expanded(
                flex: 3,
                child: Text(
                  updatedAt,
                  style: const TextStyle(fontSize: 13, color: Colors.black87),
                ),
              ),
            ],
          ),
        ),
        if (!isLast)
          const Divider(
            height: 1,
            color: Color(0xFFEEEEEE),
            indent: 16,
            endIndent: 16,
          ),
      ],
    );
  }

  /// 格式化数字，添加千位分隔符
  String _formatNumber(int number) {
    return number.toString().replaceAllMapped(
      RegExp(r'(\d{1,3})(?=(\d{3})+(?!\d))'),
      (Match m) => '${m[1]},',
    );
  }
}

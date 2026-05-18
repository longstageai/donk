package builtin

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"

	"github.com/longstageai/donk/donk/internal/tool"
)

// Calculator 计算器工具
// 用于执行数学计算，支持复杂表达式和多种数学函数
// 特性：
// - 支持基本运算：加减乘除、幂运算、取模
// - 支持数学函数：sin, cos, tan, sqrt, log, ln, abs, floor, ceil, round
// - 支持常量：pi, e
// - 支持括号嵌套
// - 安全的表达式求值（防止代码注入）

type Calculator struct {
	maxExpressionLength int // 最大表达式长度
	maxPrecision        int // 最大小数精度
}

// CalculatorOption 计算器配置选项
type CalculatorOption func(*Calculator)

// WithMaxExpressionLength 设置最大表达式长度
func WithMaxExpressionLength(length int) CalculatorOption {
	return func(c *Calculator) {
		c.maxExpressionLength = length
	}
}

// WithMaxPrecision 设置最大小数精度
func WithMaxPrecision(precision int) CalculatorOption {
	return func(c *Calculator) {
		c.maxPrecision = precision
	}
}

// NewCalculator 创建计算器工具
func NewCalculator(opts ...CalculatorOption) *Calculator {
	c := &Calculator{
		maxExpressionLength: 1000, // 默认最大 1000 字符
		maxPrecision:        10,   // 默认 10 位小数
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Name 返回工具名称
func (c *Calculator) Name() string {
	return "calculator"
}

// Description 返回工具描述
func (c *Calculator) Description() string {
	return "执行数学计算，支持加减乘除、幂运算、三角函数、对数、绝对值等。支持括号嵌套和常量 pi、e。"
}

// Version 返回版本
func (c *Calculator) Version() string {
	return "1.1.0"
}

// Category 返回分类
func (c *Calculator) Category() string {
	return string(tool.CategoryCompute)
}

// Parameters 返回参数定义
func (c *Calculator) Parameters() *tool.Schema {
	schema := tool.NewSchema()
	schema.Properties = map[string]*tool.Property{
		"expression": {
			Type:        "string",
			Description: "数学表达式，例如: 1+2*3, sin(pi/2), sqrt(16), 2^10",
		},
		"precision": {
			Type:        "integer",
			Description: "结果小数精度（0-15），默认自动",
			Default:     0,
		},
	}
	schema.Required = []string{"expression"}
	return schema
}

// Execute 执行计算
func (c *Calculator) Execute(ctx *tool.Context) (*tool.Result, error) {
	// 获取表达式
	expr, ok := ctx.Params["expression"].(string)
	if !ok || expr == "" {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams, "表达式不能为空"), nil
	}

	// 检查表达式长度
	if len(expr) > c.maxExpressionLength {
		return tool.NewErrorResultWithMsg(tool.ErrCodeInvalidParams,
			fmt.Sprintf("表达式长度 %d 超过限制 %d", len(expr), c.maxExpressionLength)), nil
	}

	// 获取精度
	precision := c.maxPrecision
	if p, ok := ctx.Params["precision"].(float64); ok && p > 0 {
		precision = int(p)
		if precision > 15 {
			precision = 15
		}
	}

	// 解析并计算表达式
	result, err := c.evaluateExpression(expr)
	if err != nil {
		return tool.NewErrorResultWithMsg(tool.ErrCodeExecution, fmt.Sprintf("计算错误: %v", err)), nil
	}

	// 格式化结果
	formattedResult := c.formatResult(result, precision)

	return tool.NewResult(map[string]any{
		"expression": expr,
		"result":     result,
		"formatted":  formattedResult,
	}), nil
}

// evaluateExpression 表达式求值主函数
func (c *Calculator) evaluateExpression(expr string) (float64, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return 0, fmt.Errorf("表达式为空")
	}

	// 预处理表达式
	expr = c.preprocessExpression(expr)

	// 验证表达式安全性
	if err := c.validateExpression(expr); err != nil {
		return 0, err
	}

	// 解析并计算
	parser := &expressionParser{expression: expr}
	return parser.parse()
}

// preprocessExpression 预处理表达式
func (c *Calculator) preprocessExpression(expr string) string {
	expr = strings.ToLower(expr)

	// 替换常量
	expr = strings.ReplaceAll(expr, "pi", fmt.Sprintf("%v", math.Pi))
	expr = strings.ReplaceAll(expr, "e", fmt.Sprintf("%v", math.E))

	// 替换幂运算符
	expr = strings.ReplaceAll(expr, "^", "**")

	return expr
}

// validateExpression 验证表达式安全性
func (c *Calculator) validateExpression(expr string) error {
	// 只允许特定字符
	allowedChars := "0123456789.+-*/%()[]{} 	\n"
	allowedFuncs := []string{"sin", "cos", "tan", "sqrt", "log", "ln", "abs", "floor", "ceil", "round", "max", "min", "pow"}

	// 检查非法字符
	for _, ch := range expr {
		if !strings.ContainsRune(allowedChars, ch) && !unicode.IsLetter(ch) {
			return fmt.Errorf("表达式包含非法字符: %c", ch)
		}
	}

	// 检查括号匹配
	if !c.checkBrackets(expr) {
		return fmt.Errorf("括号不匹配")
	}

	// 检查函数名合法性
	tempExpr := expr
	for _, fn := range allowedFuncs {
		tempExpr = strings.ReplaceAll(tempExpr, fn, "")
	}

	// 检查剩余字母（非法函数名）
	for _, ch := range tempExpr {
		if unicode.IsLetter(ch) {
			return fmt.Errorf("非法函数或变量: %c", ch)
		}
	}

	return nil
}

// checkBrackets 检查括号匹配
func (c *Calculator) checkBrackets(expr string) bool {
	stack := []rune{}
	pairs := map[rune]rune{')': '(', ']': '[', '}': '{'}

	for _, ch := range expr {
		switch ch {
		case '(', '[', '{':
			stack = append(stack, ch)
		case ')', ']', '}':
			if len(stack) == 0 {
				return false
			}
			if stack[len(stack)-1] != pairs[ch] {
				return false
			}
			stack = stack[:len(stack)-1]
		}
	}

	return len(stack) == 0
}

// formatResult 格式化结果
func (c *Calculator) formatResult(result float64, precision int) string {
	// 检查是否为整数
	if result == math.Trunc(result) {
		return strconv.FormatInt(int64(result), 10)
	}

	// 格式化小数
	format := fmt.Sprintf("%%.%df", precision)
	formatted := fmt.Sprintf(format, result)

	// 去除末尾的0
	formatted = strings.TrimRight(formatted, "0")
	formatted = strings.TrimRight(formatted, ".")

	return formatted
}

// expressionParser 表达式解析器
type expressionParser struct {
	expression string
	pos        int
}

// parse 开始解析
func (p *expressionParser) parse() (float64, error) {
	return p.parseExpression()
}

// parseExpression 解析表达式（处理加减）
func (p *expressionParser) parseExpression() (float64, error) {
	left, err := p.parseTerm()
	if err != nil {
		return 0, err
	}

	for {
		p.skipWhitespace()
		if p.pos >= len(p.expression) {
			break
		}

		ch := p.expression[p.pos]
		if ch != '+' && ch != '-' {
			break
		}

		p.pos++
		right, err := p.parseTerm()
		if err != nil {
			return 0, err
		}

		if ch == '+' {
			left += right
		} else {
			left -= right
		}
	}

	return left, nil
}

// parseTerm 解析项（处理乘除模）
func (p *expressionParser) parseTerm() (float64, error) {
	left, err := p.parseFactor()
	if err != nil {
		return 0, err
	}

	for {
		p.skipWhitespace()
		if p.pos >= len(p.expression) {
			break
		}

		ch := p.expression[p.pos]
		if ch != '*' && ch != '/' && ch != '%' {
			break
		}

		p.pos++
		right, err := p.parseFactor()
		if err != nil {
			return 0, err
		}

		switch ch {
		case '*':
			left *= right
		case '/':
			if right == 0 {
				return 0, fmt.Errorf("除数不能为零")
			}
			left /= right
		case '%':
			left = float64(int(left) % int(right))
		}
	}

	return left, nil
}

// parseFactor 解析因子（处理幂运算）
func (p *expressionParser) parseFactor() (float64, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return 0, err
	}

	for {
		p.skipWhitespace()
		if p.pos >= len(p.expression)-1 {
			break
		}

		if p.expression[p.pos] != '*' || p.expression[p.pos+1] != '*' {
			break
		}

		p.pos += 2
		right, err := p.parsePrimary()
		if err != nil {
			return 0, err
		}

		left = math.Pow(left, right)
	}

	return left, nil
}

// parsePrimary 解析基本元素（数字、括号、函数）
func (p *expressionParser) parsePrimary() (float64, error) {
	p.skipWhitespace()

	if p.pos >= len(p.expression) {
		return 0, fmt.Errorf("意外的表达式结束")
	}

	ch := p.expression[p.pos]

	// 处理括号
	if ch == '(' {
		p.pos++
		value, err := p.parseExpression()
		if err != nil {
			return 0, err
		}
		p.skipWhitespace()
		if p.pos >= len(p.expression) || p.expression[p.pos] != ')' {
			return 0, fmt.Errorf("缺少右括号")
		}
		p.pos++
		return value, nil
	}

	// 处理函数调用
	if unicode.IsLetter(rune(ch)) {
		return p.parseFunction()
	}

	// 处理数字
	return p.parseNumber()
}

// parseFunction 解析函数调用
func (p *expressionParser) parseFunction() (float64, error) {
	start := p.pos
	for p.pos < len(p.expression) && unicode.IsLetter(rune(p.expression[p.pos])) {
		p.pos++
	}

	funcName := p.expression[start:p.pos]
	p.skipWhitespace()

	// 检查是否有括号
	if p.pos >= len(p.expression) || p.expression[p.pos] != '(' {
		return 0, fmt.Errorf("函数调用需要括号: %s", funcName)
	}

	p.pos++ // 跳过左括号
	arg, err := p.parseExpression()
	if err != nil {
		return 0, err
	}

	p.skipWhitespace()
	if p.pos >= len(p.expression) || p.expression[p.pos] != ')' {
		return 0, fmt.Errorf("函数调用缺少右括号: %s", funcName)
	}
	p.pos++ // 跳过右括号

	// 执行函数
	switch funcName {
	case "sin":
		return math.Sin(arg), nil
	case "cos":
		return math.Cos(arg), nil
	case "tan":
		return math.Tan(arg), nil
	case "sqrt":
		if arg < 0 {
			return 0, fmt.Errorf("不能对负数开平方")
		}
		return math.Sqrt(arg), nil
	case "log", "ln":
		if arg <= 0 {
			return 0, fmt.Errorf("对数函数的参数必须大于0")
		}
		return math.Log(arg), nil
	case "log10":
		if arg <= 0 {
			return 0, fmt.Errorf("对数函数的参数必须大于0")
		}
		return math.Log10(arg), nil
	case "abs":
		return math.Abs(arg), nil
	case "floor":
		return math.Floor(arg), nil
	case "ceil":
		return math.Ceil(arg), nil
	case "round":
		return math.Round(arg), nil
	default:
		return 0, fmt.Errorf("未知函数: %s", funcName)
	}
}

// parseNumber 解析数字
func (p *expressionParser) parseNumber() (float64, error) {
	p.skipWhitespace()

	start := p.pos
	hasDot := false

	// 处理负号
	if p.pos < len(p.expression) && p.expression[p.pos] == '-' {
		p.pos++
	}

	for p.pos < len(p.expression) {
		ch := p.expression[p.pos]
		if ch >= '0' && ch <= '9' {
			p.pos++
		} else if ch == '.' && !hasDot {
			hasDot = true
			p.pos++
		} else {
			break
		}
	}

	if start == p.pos {
		return 0, fmt.Errorf("在位置 %d 期望数字", p.pos)
	}

	numStr := p.expression[start:p.pos]
	value, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("无效的数字: %s", numStr)
	}

	return value, nil
}

// skipWhitespace 跳过空白字符
func (p *expressionParser) skipWhitespace() {
	for p.pos < len(p.expression) {
		ch := p.expression[p.pos]
		if ch != ' ' && ch != '\t' && ch != '\n' && ch != '\r' {
			break
		}
		p.pos++
	}
}

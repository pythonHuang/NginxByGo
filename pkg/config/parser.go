package config

// Parser 解析器
type Parser struct {
	scanner *Scanner
	tokens  []Token
	pos     int
}

// NewParser 创建解析器
func NewParser(content []byte) *Parser {
	scanner := NewScanner(content)
	tokens := make([]Token, 0)

	for {
		tok := scanner.Next()
		tokens = append(tokens, tok)
		if tok.Type == TokenEOF {
			break
		}
	}

	return &Parser{
		scanner: scanner,
		tokens:   tokens,
		pos:      0,
	}
}

// Parse 解析配置
func (p *Parser) Parse() (*Config, error) {
	cfg := DefaultConfig()

	// 解析配置块
	for p.pos < len(p.tokens) {
		tok := p.peek()

		if tok.Type == TokenEOF {
			break
		}

		if tok.Type != TokenIdent {
			p.error("unexpected token: %v", tok.Value)
			p.next()
			continue
		}

		// 解析 directive
		if err := p.parseDirective(cfg); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

func (p *Parser) parseDirective(cfg *Config) error {
	name := p.next().Value

	switch name {
	case "events":
		return p.parseEventsBlock(&cfg.Events)
	case "http":
		return p.parseHTTPBlock(&cfg.HTTP)
	case "worker_processes":
		if p.peek().Type == TokenIdent {
			val := p.next()
			if val.Value == "auto" {
				cfg.Global.WorkerProcesses = 0
			}
		} else if p.peek().Type == TokenString {
			val := p.next()
			cfg.Global.WorkerProcesses = atoi(val.Value)
		}
		p.expect(TokenSemicolon)
	case "error_log":
		val := p.next()
		cfg.Global.ErrorLog = val.Value
		p.expect(TokenSemicolon)
	case "pid":
		val := p.next()
		cfg.Global.PIDFile = val.Value
		p.expect(TokenSemicolon)
	default:
		return p.skipBlock()
	}

	return nil
}

func (p *Parser) parseEventsBlock(cfg *EventsConfig) error {
	p.expect(TokenLeftBrace)

	for {
		tok := p.peek()
		if tok.Type == TokenRightBrace {
			p.next()
			break
		}

		name := p.next().Value
		switch name {
		case "worker_connections":
			val := p.next()
			cfg.WorkerConnections = atoi(val.Value)
		case "multi_accept":
			val := p.next()
			cfg.MultiAccept = val.Value == "on"
		case "use":
			val := p.next()
			cfg.Use = val.Value
		}
		p.expect(TokenSemicolon)
	}

	return nil
}

func (p *Parser) parseHTTPBlock(cfg *HTTPConfig) error {
	p.expect(TokenLeftBrace)

	for {
		tok := p.peek()
		if tok.Type == TokenRightBrace {
			p.next()
			break
		}

		name := p.next().Value
		switch name {
		case "server":
			server, err := p.parseServerBlock()
			if err != nil {
				return err
			}
			cfg.Servers = append(cfg.Servers, server)
		case "upstream":
			upstream, err := p.parseUpstreamBlock()
			if err != nil {
				return err
			}
			cfg.Upstreams = append(cfg.Upstreams, upstream)
		case "include":
			p.next()
			p.expect(TokenSemicolon)
		default:
			p.skipBlock()
		}
	}

	return nil
}

func (p *Parser) parseServerBlock() (*ServerConfig, error) {
	server := &ServerConfig{
		Listen:     make([]string, 0),
		ServerName: make([]string, 0),
		Index:      []string{"index.html"},
		Root:       "html",
	}

	p.expect(TokenLeftBrace)

	for {
		tok := p.peek()
		if tok.Type == TokenRightBrace {
			p.next()
			break
		}

		name := p.next().Value
		switch name {
		case "listen":
			val := p.next()
			server.Listen = append(server.Listen, val.Value)
			p.expect(TokenSemicolon)
		case "server_name":
			for {
				tok := p.peek()
				if tok.Type == TokenSemicolon {
					break
				}
				val := p.next()
				server.ServerName = append(server.ServerName, val.Value)
			}
			p.expect(TokenSemicolon)
		case "root":
			val := p.next()
			server.Root = val.Value
			p.expect(TokenSemicolon)
		case "location":
			loc, err := p.parseLocationBlock()
			if err != nil {
				return nil, err
			}
			server.Locations = append(server.Locations, loc)
		default:
			p.skipBlock()
		}
	}

	return server, nil
}

func (p *Parser) parseLocationBlock() (*LocationConfig, error) {
	loc := &LocationConfig{
		Index: []string{"index.html"},
	}

	// 解析 location 路径
	pathTok := p.next()
	loc.Path = pathTok.Value

	// 检查修饰符
	if len(loc.Path) > 0 && loc.Path[0] == '=' {
		loc.Modifier = "="
		loc.Path = loc.Path[1:]
	} else if len(loc.Path) > 2 && loc.Path[:2] == "^~" {
		loc.Modifier = "^~"
		loc.Path = loc.Path[2:]
	}

	p.expect(TokenLeftBrace)

	for {
		tok := p.peek()
		if tok.Type == TokenRightBrace {
			p.next()
			break
		}

		name := p.next().Value
		switch name {
		case "root":
			val := p.next()
			loc.Root = val.Value
			p.expect(TokenSemicolon)
		case "proxy_pass":
			val := p.next()
			loc.ProxyPass = val.Value
			p.expect(TokenSemicolon)
		case "index":
			for {
				tok := p.peek()
				if tok.Type == TokenSemicolon {
					break
				}
				val := p.next()
				loc.Index = append(loc.Index, val.Value)
			}
			p.expect(TokenSemicolon)
		default:
			p.skipBlock()
		}
	}

	return loc, nil
}

func (p *Parser) parseUpstreamBlock() (*UpstreamConfig, error) {
	upstream := &UpstreamConfig{
		Servers: make([]*UpstreamServer, 0),
	}

	// 解析 upstream 名称
	nameTok := p.next()
	upstream.Name = nameTok.Value

	p.expect(TokenLeftBrace)

	for {
		tok := p.peek()
		if tok.Type == TokenRightBrace {
			p.next()
			break
		}

		name := p.next().Value
		if name == "server" {
			server := &UpstreamServer{}
			addrTok := p.next()
			server.Address = addrTok.Value
			// TODO: 解析权重等参数
			upstream.Servers = append(upstream.Servers, server)
			p.expect(TokenSemicolon)
		}
	}

	return upstream, nil
}

func (p *Parser) next() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	t := p.tokens[p.pos]
	p.pos++
	return t
}

func (p *Parser) peek() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) expect(t TokenType) {
	tok := p.next()
	if tok.Type != t {
		p.error("expected %v but got %v", t, tok.Type)
	}
}

func (p *Parser) error(format string, args ...interface{}) {
	// 简单的错误输出
}

func (p *Parser) skipBlock() error {
	for {
		tok := p.peek()
		if tok.Type == TokenEOF {
			break
		}
		if tok.Type == TokenLeftBrace {
			p.next()
			p.skipBlock()
		}
		if tok.Type == TokenRightBrace {
			p.next()
			break
		}
		p.next()
	}
	return nil
}
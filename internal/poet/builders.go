package poet

import "github.com/asavoy/alt-sqlc-gen-python/internal/ast"

func Alias(name string) *ast.Node {
	return &ast.Node{
		Node: &ast.Node_Alias{
			Alias: &ast.Alias{
				Name: name,
			},
		},
	}
}

func Await(value *ast.Node) *ast.Node {
	return &ast.Node{
		Node: &ast.Node_Await{
			Await: &ast.Await{
				Value: value,
			},
		},
	}
}

func Attribute(value *ast.Node, attr string) *ast.Node {
	return &ast.Node{
		Node: &ast.Node_Attribute{
			Attribute: &ast.Attribute{
				Value: value,
				Attr:  attr,
			},
		},
	}
}

func Comment(text string) *ast.Node {
	return &ast.Node{
		Node: &ast.Node_Comment{
			Comment: &ast.Comment{
				Text: text,
			},
		},
	}
}

func Expr(value *ast.Node) *ast.Node {
	return &ast.Node{
		Node: &ast.Node_Expr{
			Expr: &ast.Expr{
				Value: value,
			},
		},
	}
}

func Is() *ast.Node {
	return &ast.Node{
		Node: &ast.Node_Is{
			Is: &ast.Is{},
		},
	}
}

func Name(id string) *ast.Node {
	return &ast.Node{
		Node: &ast.Node_Name{
			Name: &ast.Name{Id: id},
		},
	}
}

func BitOr() *ast.Node {
	return &ast.Node{
		Node: &ast.Node_BitOr{
			BitOr: &ast.BitOr{},
		},
	}
}

func BinOp(left *ast.Node, op *ast.Node, right *ast.Node) *ast.Node {
	return &ast.Node{
		Node: &ast.Node_BinOp{
			BinOp: &ast.BinOp{
				Left:  left,
				Op:    op,
				Right: right,
			},
		},
	}
}

func Return(value *ast.Node) *ast.Node {
	return &ast.Node{
		Node: &ast.Node_Return{
			Return: &ast.Return{
				Value: value,
			},
		},
	}
}

func Yield(value *ast.Node) *ast.Node {
	return &ast.Node{
		Node: &ast.Node_Yield{
			Yield: &ast.Yield{
				Value: value,
			},
		},
	}
}

func TypeVar(name string, bound *ast.Node) *ast.Node {
	return &ast.Node{
		Node: &ast.Node_TypeVar{
			TypeVar: &ast.TypeVar{
				Name:  name,
				Bound: bound,
			},
		},
	}
}

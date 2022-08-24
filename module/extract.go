package module

import (
	"strings"

	"github.com/fatih/structtag"
	pgs "github.com/lyft/protoc-gen-star"
	pgsgo "github.com/lyft/protoc-gen-star/lang/go"
	"github.com/storskegg/protoc-gen-gotag/tagger"
)

const (
	omitEmptyStr = "-with-omitempty"
)

type autoAddTagsTransformer struct {
	Omitempty bool
	NameFunc  func(name pgs.Name) pgs.Name
}

type tagExtractor struct {
	pgs.Visitor
	pgs.DebuggerCommon
	pgsgo.Context

	tags        map[string]map[string]*structtag.Tags
	autoAddTags map[string]*autoAddTagsTransformer
}

func newTagExtractor(d pgs.DebuggerCommon, ctx pgsgo.Context, autoTags []string) *tagExtractor {
	v := &tagExtractor{DebuggerCommon: d, Context: ctx, autoAddTags: map[string]*autoAddTagsTransformer{}}
	v.Visitor = pgs.PassThroughVisitor(v)
	var tagOrig string
	for _, autoTag := range autoTags {
		info := strings.Split(autoTag, "-as-")
		tagName := info[0]
		tagOrig = tagName

		omitempty := strings.HasSuffix(tagName, omitEmptyStr)

		if omitempty {
			tagName = strings.TrimSuffix(tagName, omitEmptyStr)
		}

		if len(info) == 1 {
			v.autoAddTags[tagName] = &autoAddTagsTransformer{
				Omitempty: omitempty,
				NameFunc:  pgs.Name.LowerSnakeCase,
			}
		} else {
			switch strings.ToLower(info[1]) {
			case "lower_snake", "lower_snake_case", "snake", "snake_case":
				v.autoAddTags[tagName] = &autoAddTagsTransformer{
					Omitempty: omitempty,
					NameFunc:  pgs.Name.LowerSnakeCase,
				}
			case "upper_snake", "upper_snake_case":
				v.autoAddTags[tagName] = &autoAddTagsTransformer{
					Omitempty: omitempty,
					NameFunc:  pgs.Name.UpperSnakeCase,
				}
			case "lower_camel", "lower_camel_case", "camel", "camel_case":
				v.autoAddTags[tagName] = &autoAddTagsTransformer{
					Omitempty: omitempty,
					NameFunc:  pgs.Name.LowerCamelCase,
				}
			case "upper_camel", "upper_camel_case":
				v.autoAddTags[tagName] = &autoAddTagsTransformer{
					Omitempty: omitempty,
					NameFunc:  pgs.Name.UpperCamelCase,
				}
			case "dot_notation", "dot", "lower_dot_notation", "lower_dot":
				v.autoAddTags[tagName] = &autoAddTagsTransformer{
					Omitempty: omitempty,
					NameFunc:  pgs.Name.LowerDotNotation,
				}
			case "upper_dot", "upper_dot_notation":
				v.autoAddTags[tagName] = &autoAddTagsTransformer{
					Omitempty: omitempty,
					NameFunc:  pgs.Name.UpperDotNotation,
				}
			}
		}

		if omitempty {
			delete(v.autoAddTags, tagOrig)
		}
	}
	return v
}

func (v *tagExtractor) VisitOneOf(o pgs.OneOf) (pgs.Visitor, error) {
	var tval string
	ok, err := o.Extension(tagger.E_OneofTags, &tval)
	if err != nil {
		return nil, err
	}

	msgName := v.Context.Name(o.Message()).String()

	if v.tags[msgName] == nil {
		v.tags[msgName] = map[string]*structtag.Tags{}
	}

	if !ok {
		return v, nil
	}

	tags, err := structtag.Parse(tval)
	if err != nil {
		return nil, err
	}

	v.tags[msgName][v.Context.Name(o).String()] = tags

	return v, nil
}

func (v *tagExtractor) VisitField(f pgs.Field) (pgs.Visitor, error) {
	var tval string
	ok, err := f.Extension(tagger.E_Tags, &tval)
	if err != nil {
		return nil, err
	}

	msgName := v.Context.Name(f.Message()).String()
	if f.InOneOf() && !f.Descriptor().GetProto3Optional() {
		msgName = f.Message().Name().UpperCamelCase().String() + "_" + f.Name().UpperCamelCase().String()
	}

	if v.tags[msgName] == nil {
		v.tags[msgName] = map[string]*structtag.Tags{}
	}

	tags := structtag.Tags{}
	if len(v.autoAddTags) > 0 {
		for tag, transformer := range v.autoAddTags {
			v.DebuggerCommon.Log("XXX LIAM -- tag:", tag)
			v.DebuggerCommon.Debug("XXX LIAM -- tag:", tag)
			v.DebuggerCommon.Fail("XXX LIAM -- tag:", tag)

			if strings.HasSuffix(tag, omitEmptyStr) {
				continue
			}
			t := structtag.Tag{
				Key:     tag,
				Name:    transformer.NameFunc(v.Context.Name(f)).String(),
				Options: nil,
			}

			if transformer.Omitempty {
				if tag == "graphql" {
					t.Options = []string{"optional"}
				} else {
					t.Options = []string{"omitempty"}
				}
			}

			if err := tags.Set(&t); err != nil {
				v.DebuggerCommon.Fail("Error without tag", err)
			}
		}
	}

	if !ok {
		v.tags[msgName][v.Context.Name(f).String()] = &tags
		return v, nil
	}

	newTags, err := structtag.Parse(tval)
	v.CheckErr(err)
	for _, tag := range newTags.Tags() {
		if err := tags.Set(tag); err != nil {
			v.DebuggerCommon.Fail("Error with tag: ", err)
		}
	}

	v.tags[msgName][v.Context.Name(f).String()] = &tags

	return v, nil
}

func (v *tagExtractor) Extract(f pgs.File) StructTags {
	v.tags = map[string]map[string]*structtag.Tags{}

	v.CheckErr(pgs.Walk(v, f))

	return v.tags
}

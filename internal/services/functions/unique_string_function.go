package functions

import (
	"context"
	"math/bits"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type UniqueStringFunction struct{}

func (b *UniqueStringFunction) Metadata(ctx context.Context, request function.MetadataRequest, response *function.MetadataResponse) {
	response.Name = "unique_string"
}

func (b *UniqueStringFunction) Definition(ctx context.Context, request function.DefinitionRequest, response *function.DefinitionResponse) {
	response.Definition = function.Definition{
		Parameters: []function.Parameter{
			function.ListParameter{
				ElementType:         types.StringType,
				AllowNullValue:      false,
				AllowUnknownValues:  false,
				Name:                "base_string",
				Description:         "The values used in the hash function to create a unique string.",
				MarkdownDescription: "The values used in the hash function to create a unique string.",
			},
		},
		Return:              function.StringReturn{},
		Summary:             "Creates a deterministic hash string based on the values provided as parameters.",
		Description:         "This function constructs an Azure equivalent uniqueString value. It is useful for migrating existing resources based on th ARM uniqueString function.",
		MarkdownDescription: "This function constructs an Azure equivalent `uniqueString` value. It is useful for migrating existing resources based on th ARM `uniqueString` function.",
	}
}

func (b *UniqueStringFunction) Run(ctx context.Context, request function.RunRequest, response *function.RunResponse) {
	var baseString types.List

	if response.Error = request.Arguments.Get(ctx, &baseString); response.Error != nil {
		return
	}

	var slice []string
	if diagnostics := baseString.ElementsAs(ctx, &slice, false); diagnostics.HasError() {
		response.Error = function.FuncErrorFromDiags(ctx, diagnostics)
		return
	}

	uniqueString := uniqueString(slice...)

	response.Error = response.Result.Set(ctx, types.StringValue(uniqueString))
}

var _ function.Function = &UniqueStringFunction{}

func uniqueString(values ...string) string {
	value := strings.Join(values, "-")
	hash := murmurHash64(value)
	return base32Encode(hash)
}

func base32Encode(value uint64) string {
	const text = "abcdefghijklmnopqrstuvwxyz234567"
	var builder strings.Builder
	for i := 0; i < 13; i++ {
		builder.WriteByte(text[int32(value>>59)])
		value <<= 5
	}
	return builder.String()
}

func murmurHash64(value string) uint64 {
	bytes := []byte(value)
	return murmurHash64A(bytes, 0)
}

func murmurHash64A(data []byte, seed uint32) uint64 {
	length := len(data)
	h1 := seed
	h2 := seed

	var index int
	for index = 0; index+7 < length; index += 8 {
		k1 := uint32(data[index]) | uint32(data[index+1])<<8 | uint32(data[index+2])<<16 | uint32(data[index+3])<<24
		k3 := uint32(data[index+4]) | uint32(data[index+5])<<8 | uint32(data[index+6])<<16 | uint32(data[index+7])<<24
		k1 *= 597399067
		k1 = bits.RotateLeft32(k1, 15)
		k1 *= 2869860233
		h1 ^= k1
		h1 = bits.RotateLeft32(h1, 19)
		h1 += h2
		h1 = h1*5 + 1444728091
		k3 *= 2869860233
		k3 = bits.RotateLeft32(k3, 17)
		k3 *= 597399067
		h2 ^= k3
		h2 = bits.RotateLeft32(h2, 13)
		h2 += h1
		h2 = h2*5 + 197830471
	}

	if tail := length - index; tail > 0 {
		var k2 uint32

		if tail >= 4 {
			k2 = uint32(data[index]) | (uint32(data[index+1]) << 8) | (uint32(data[index+2]) << 16) | (uint32(data[index+3]) << 24)
		} else {
			switch tail {
			case 2:
				k2 = uint32(data[index]) | (uint32(data[index+1]) << 8)
			case 3:
				k2 = uint32(data[index]) | (uint32(data[index+1]) << 8) | (uint32(data[index+2]) << 16)
			default:
				k2 = uint32(data[index])
			}
		}

		k2 *= 597399067
		k2 = bits.RotateLeft32(k2, 15)
		k2 *= 2869860233
		h1 ^= k2

		if tail > 4 {
			var k4 int32
			switch tail {
			case 6:
				k4 = int32(data[index+4]) | (int32(data[index+5]) << 8)
			case 7:
				k4 = int32(data[index+4]) | (int32(data[index+5]) << 8) | (int32(data[index+6]) << 16)
			default:
				k4 = int32(data[index+4])
			}
			k4 *= -1425107063
			i4 := uint32(k4)
			i4 = bits.RotateLeft32(i4, 17)
			i4 *= 597399067
			h2 ^= i4
		}
	}

	h1 ^= uint32(length)
	h2 ^= uint32(length)
	h1 += h2
	h2 += h1
	h1 ^= h1 >> 16
	h1 *= 2246822507
	h1 ^= h1 >> 13
	h1 *= 3266489909
	h1 ^= h1 >> 16
	h2 ^= h2 >> 16
	h2 *= 2246822507
	h2 ^= h2 >> 13
	h2 *= 3266489909
	h2 ^= h2 >> 16
	h1 += h2
	h2 += h1

	return (uint64(h2) << 32) | uint64(h1)
}

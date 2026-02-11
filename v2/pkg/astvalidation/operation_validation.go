// Package astvalidation implements the validation rules specified in the GraphQL specification.
package astvalidation

import (
	"net/http"

	"github.com/wundergraph/graphql-go-tools/v2/pkg/apollocompatibility"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/astvisitor"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/errorcodes"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/operationreport"
)

// OperationValidationRuleName identifies a specific operation validation rule.
type OperationValidationRuleName string

const (
	AllVariablesUsedRule                    OperationValidationRuleName = "AllVariablesUsed"
	AllVariableUsesDefinedRule              OperationValidationRuleName = "AllVariableUsesDefined"
	DocumentContainsExecutableOperationRule OperationValidationRuleName = "DocumentContainsExecutableOperation"
	OperationNameUniquenessRule             OperationValidationRuleName = "OperationNameUniqueness"
	LoneAnonymousOperationRule              OperationValidationRuleName = "LoneAnonymousOperation"
	SubscriptionSingleRootFieldRule         OperationValidationRuleName = "SubscriptionSingleRootField"
	FieldSelectionsRule                     OperationValidationRuleName = "FieldSelections"
	FieldSelectionMergingRule               OperationValidationRuleName = "FieldSelectionMerging"
	KnownArgumentsRule                      OperationValidationRuleName = "KnownArguments"
	ValuesRule                              OperationValidationRuleName = "Values"
	ArgumentUniquenessRule                  OperationValidationRuleName = "ArgumentUniqueness"
	RequiredArgumentsRule                   OperationValidationRuleName = "RequiredArguments"
	FragmentsRule                           OperationValidationRuleName = "Fragments"
	DirectivesAreDefinedRule                OperationValidationRuleName = "DirectivesAreDefined"
	DirectivesAreInValidLocationsRule       OperationValidationRuleName = "DirectivesAreInValidLocations"
	VariableUniquenessRule                  OperationValidationRuleName = "VariableUniqueness"
	DirectivesAreUniquePerLocationRule      OperationValidationRuleName = "DirectivesAreUniquePerLocation"
	VariablesAreInputTypesRule              OperationValidationRuleName = "VariablesAreInputTypes"
)

type OperationValidatorOptions struct {
	ApolloCompatibilityFlags apollocompatibility.Flags
	DisabledRules            map[OperationValidationRuleName]struct{}
}

func WithApolloCompatibilityFlags(flags apollocompatibility.Flags) Option {
	return func(options *OperationValidatorOptions) {
		options.ApolloCompatibilityFlags = flags
	}
}

// WithDisabledRules returns an Option that disables the specified validation rules.
// Disabled rules will not be registered on the validator.
func WithDisabledRules(rules ...OperationValidationRuleName) Option {
	return func(options *OperationValidatorOptions) {
		if options.DisabledRules == nil {
			options.DisabledRules = make(map[OperationValidationRuleName]struct{}, len(rules))
		}
		for _, rule := range rules {
			options.DisabledRules[rule] = struct{}{}
		}
	}
}

type Option func(options *OperationValidatorOptions)

type namedRule struct {
	name OperationValidationRuleName
	rule Rule
}

// DefaultOperationValidator returns a fully initialized OperationValidator with all default rules registered.
// Use WithDisabledRules to skip specific rules.
func DefaultOperationValidator(options ...Option) *OperationValidator {
	var opts OperationValidatorOptions
	for _, opt := range options {
		opt(&opts)
	}
	validator := OperationValidator{
		walker: astvisitor.NewWalkerWithID(48, "OperationValidator"),
	}

	if opts.ApolloCompatibilityFlags.UseGraphQLValidationFailedStatus {
		validator.walker.OnExternalError = func(err *operationreport.ExternalError) {
			err.ExtensionCode = errorcodes.GraphQLValidationFailed
			err.StatusCode = http.StatusBadRequest
		}
	}

	defaultRules := []namedRule{
		{AllVariablesUsedRule, AllVariablesUsed()},
		{AllVariableUsesDefinedRule, AllVariableUsesDefined()},
		{DocumentContainsExecutableOperationRule, DocumentContainsExecutableOperation()},
		{OperationNameUniquenessRule, OperationNameUniqueness()},
		{LoneAnonymousOperationRule, LoneAnonymousOperation()},
		{SubscriptionSingleRootFieldRule, SubscriptionSingleRootField()},
		{FieldSelectionsRule, FieldSelections()},
		{FieldSelectionMergingRule, FieldSelectionMerging()},
		{KnownArgumentsRule, KnownArguments()},
		{ValuesRule, Values()},
		{ArgumentUniquenessRule, ArgumentUniqueness()},
		{RequiredArgumentsRule, RequiredArguments()},
		{FragmentsRule, Fragments()},
		{DirectivesAreDefinedRule, DirectivesAreDefined()},
		{DirectivesAreInValidLocationsRule, DirectivesAreInValidLocations()},
		{VariableUniquenessRule, VariableUniqueness()},
		{DirectivesAreUniquePerLocationRule, DirectivesAreUniquePerLocation()},
		{VariablesAreInputTypesRule, VariablesAreInputTypes()},
	}

	for _, nr := range defaultRules {
		if _, disabled := opts.DisabledRules[nr.name]; disabled {
			continue
		}
		validator.RegisterRule(nr.rule)
	}

	return &validator
}

func NewOperationValidator(rules []Rule) *OperationValidator {
	validator := OperationValidator{
		walker: astvisitor.NewWalkerWithID(48, "OperationValidator"),
	}

	for _, rule := range rules {
		validator.RegisterRule(rule)
	}

	return &validator
}

// OperationValidator orchestrates the validation process of Operations
type OperationValidator struct {
	walker astvisitor.Walker
}

// RegisterRule registers a rule to the OperationValidator
func (o *OperationValidator) RegisterRule(rule Rule) {
	rule(&o.walker)
}

// Validate validates the operation against the definition using the registered ruleset.
func (o *OperationValidator) Validate(operation, definition *ast.Document, report *operationreport.Report) ValidationState {

	if report == nil {
		report = &operationreport.Report{}
	}

	o.walker.Walk(operation, definition, report)

	if report.HasErrors() {
		return Invalid
	}
	return Valid
}

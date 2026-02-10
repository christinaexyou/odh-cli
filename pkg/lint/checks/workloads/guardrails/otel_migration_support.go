package guardrails

import (
	"context"
	"errors"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/odh-cli/pkg/lint/check"
	"github.com/lburgazzoli/odh-cli/pkg/lint/check/result"
	"github.com/lburgazzoli/odh-cli/pkg/lint/checks/shared/validate"
	"github.com/lburgazzoli/odh-cli/pkg/util/jq"
)

// hasDeprecatedOtelFields returns true if the object contains a non-empty otelExporter section.
// An empty otelExporter ({}) is not considered deprecated since there is no configuration to migrate.
func hasDeprecatedOtelFields(obj *unstructured.Unstructured) (bool, error) {
	val, err := jq.Query[map[string]any](obj, ".spec.otelExporter")

	switch {
	case errors.Is(err, jq.ErrNotFound):
		return false, nil
	case err != nil:
		return false, err
	default:
		return len(val) > 0, nil
	}
}

func (c *OtelMigrationCheck) newOtelMigrationCondition(
	_ context.Context,
	req *validate.WorkloadRequest[*unstructured.Unstructured],
) ([]result.Condition, error) {
	count := len(req.Items)

	if count == 0 {
		return []result.Condition{check.NewCondition(
			ConditionTypeOtelConfigCompatible,
			metav1.ConditionTrue,
			check.WithReason(check.ReasonVersionCompatible),
			check.WithMessage("No GuardrailsOrchestrators found using deprecated otelExporter fields"),
		)}, nil
	}

	refs := make([]string, 0, count)
	for _, obj := range req.Items {
		refs = append(refs, obj.GetNamespace()+"/"+obj.GetName())
	}
	msg := fmt.Sprintf("Found %d GuardrailsOrchestrator(s) using deprecated otelExporter fields - migrate to new format before upgrading:\n\n%s",
		count, strings.Join(refs, "\n\n"))

	return []result.Condition{check.NewCondition(
		ConditionTypeOtelConfigCompatible,
		metav1.ConditionFalse,
		check.WithReason(check.ReasonConfigurationInvalid),
		check.WithMessage("%s", msg),
		check.WithImpact(result.ImpactAdvisory),
	)}, nil
}

from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.workflow_action_definition import WorkflowActionDefinition
    from ..models.workflow_gate_definition import WorkflowGateDefinition
    from ..models.workflow_question_policy import WorkflowQuestionPolicy


T = TypeVar("T", bound="WorkflowDefinition")


@_attrs_define
class WorkflowDefinition:
    """
    Attributes:
        actions (list[WorkflowActionDefinition] | None):
        gates (list[WorkflowGateDefinition] | None):
        id (str):
        questions (WorkflowQuestionPolicy):
        stages (list[str] | None):
        version (int):
    """

    actions: list[WorkflowActionDefinition] | None
    gates: list[WorkflowGateDefinition] | None
    id: str
    questions: WorkflowQuestionPolicy
    stages: list[str] | None
    version: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        actions: list[dict[str, Any]] | None
        if isinstance(self.actions, list):
            actions = []
            for actions_type_0_item_data in self.actions:
                actions_type_0_item = actions_type_0_item_data.to_dict()
                actions.append(actions_type_0_item)

        else:
            actions = self.actions

        gates: list[dict[str, Any]] | None
        if isinstance(self.gates, list):
            gates = []
            for gates_type_0_item_data in self.gates:
                gates_type_0_item = gates_type_0_item_data.to_dict()
                gates.append(gates_type_0_item)

        else:
            gates = self.gates

        id = self.id

        questions = self.questions.to_dict()

        stages: list[str] | None
        if isinstance(self.stages, list):
            stages = self.stages

        else:
            stages = self.stages

        version = self.version

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "actions": actions,
                "gates": gates,
                "id": id,
                "questions": questions,
                "stages": stages,
                "version": version,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.workflow_action_definition import WorkflowActionDefinition
        from ..models.workflow_gate_definition import WorkflowGateDefinition
        from ..models.workflow_question_policy import WorkflowQuestionPolicy

        d = dict(src_dict)

        def _parse_actions(data: object) -> list[WorkflowActionDefinition] | None:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                actions_type_0 = []
                _actions_type_0 = data
                for actions_type_0_item_data in _actions_type_0:
                    actions_type_0_item = WorkflowActionDefinition.from_dict(
                        actions_type_0_item_data
                    )

                    actions_type_0.append(actions_type_0_item)

                return actions_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[WorkflowActionDefinition] | None, data)

        actions = _parse_actions(d.pop("actions"))

        def _parse_gates(data: object) -> list[WorkflowGateDefinition] | None:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                gates_type_0 = []
                _gates_type_0 = data
                for gates_type_0_item_data in _gates_type_0:
                    gates_type_0_item = WorkflowGateDefinition.from_dict(
                        gates_type_0_item_data
                    )

                    gates_type_0.append(gates_type_0_item)

                return gates_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[WorkflowGateDefinition] | None, data)

        gates = _parse_gates(d.pop("gates"))

        id = d.pop("id")

        questions = WorkflowQuestionPolicy.from_dict(d.pop("questions"))

        def _parse_stages(data: object) -> list[str] | None:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                stages_type_0 = cast(list[str], data)

                return stages_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[str] | None, data)

        stages = _parse_stages(d.pop("stages"))

        version = d.pop("version")

        workflow_definition = cls(
            actions=actions,
            gates=gates,
            id=id,
            questions=questions,
            stages=stages,
            version=version,
        )

        workflow_definition.additional_properties = d
        return workflow_definition

    @property
    def additional_keys(self) -> list[str]:
        return list(self.additional_properties.keys())

    def __getitem__(self, key: str) -> Any:
        return self.additional_properties[key]

    def __setitem__(self, key: str, value: Any) -> None:
        self.additional_properties[key] = value

    def __delitem__(self, key: str) -> None:
        del self.additional_properties[key]

    def __contains__(self, key: str) -> bool:
        return key in self.additional_properties

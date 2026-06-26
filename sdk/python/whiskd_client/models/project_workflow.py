from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.transition_rule import TransitionRule
    from ..models.workflow_stage import WorkflowStage


T = TypeVar("T", bound="ProjectWorkflow")


@_attrs_define
class ProjectWorkflow:
    """
    Attributes:
        id (str):
        name (str):
        stages (list[WorkflowStage] | None):
        template_id (str):
        transition_rules (list[TransitionRule] | None):
        definition_hash (str | Unset):
        definition_id (str | Unset):
        definition_version (int | Unset):
    """

    id: str
    name: str
    stages: list[WorkflowStage] | None
    template_id: str
    transition_rules: list[TransitionRule] | None
    definition_hash: str | Unset = UNSET
    definition_id: str | Unset = UNSET
    definition_version: int | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        id = self.id

        name = self.name

        stages: list[dict[str, Any]] | None
        if isinstance(self.stages, list):
            stages = []
            for stages_type_0_item_data in self.stages:
                stages_type_0_item = stages_type_0_item_data.to_dict()
                stages.append(stages_type_0_item)

        else:
            stages = self.stages

        template_id = self.template_id

        transition_rules: list[dict[str, Any]] | None
        if isinstance(self.transition_rules, list):
            transition_rules = []
            for transition_rules_type_0_item_data in self.transition_rules:
                transition_rules_type_0_item = (
                    transition_rules_type_0_item_data.to_dict()
                )
                transition_rules.append(transition_rules_type_0_item)

        else:
            transition_rules = self.transition_rules

        definition_hash = self.definition_hash

        definition_id = self.definition_id

        definition_version = self.definition_version

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "id": id,
                "name": name,
                "stages": stages,
                "templateId": template_id,
                "transitionRules": transition_rules,
            }
        )
        if definition_hash is not UNSET:
            field_dict["definitionHash"] = definition_hash
        if definition_id is not UNSET:
            field_dict["definitionId"] = definition_id
        if definition_version is not UNSET:
            field_dict["definitionVersion"] = definition_version

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transition_rule import TransitionRule
        from ..models.workflow_stage import WorkflowStage

        d = dict(src_dict)
        id = d.pop("id")

        name = d.pop("name")

        def _parse_stages(data: object) -> list[WorkflowStage] | None:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                stages_type_0 = []
                _stages_type_0 = data
                for stages_type_0_item_data in _stages_type_0:
                    stages_type_0_item = WorkflowStage.from_dict(
                        stages_type_0_item_data
                    )

                    stages_type_0.append(stages_type_0_item)

                return stages_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[WorkflowStage] | None, data)

        stages = _parse_stages(d.pop("stages"))

        template_id = d.pop("templateId")

        def _parse_transition_rules(data: object) -> list[TransitionRule] | None:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                transition_rules_type_0 = []
                _transition_rules_type_0 = data
                for transition_rules_type_0_item_data in _transition_rules_type_0:
                    transition_rules_type_0_item = TransitionRule.from_dict(
                        transition_rules_type_0_item_data
                    )

                    transition_rules_type_0.append(transition_rules_type_0_item)

                return transition_rules_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[TransitionRule] | None, data)

        transition_rules = _parse_transition_rules(d.pop("transitionRules"))

        definition_hash = d.pop("definitionHash", UNSET)

        definition_id = d.pop("definitionId", UNSET)

        definition_version = d.pop("definitionVersion", UNSET)

        project_workflow = cls(
            id=id,
            name=name,
            stages=stages,
            template_id=template_id,
            transition_rules=transition_rules,
            definition_hash=definition_hash,
            definition_id=definition_id,
            definition_version=definition_version,
        )

        project_workflow.additional_properties = d
        return project_workflow

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

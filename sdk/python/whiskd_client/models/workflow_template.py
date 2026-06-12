from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.transition_rule import TransitionRule
    from ..models.workflow_stage import WorkflowStage


T = TypeVar("T", bound="WorkflowTemplate")


@_attrs_define
class WorkflowTemplate:
    """
    Attributes:
        created_at (datetime.datetime):
        id (str):
        name (str):
        source (str):
        stages (list[WorkflowStage] | None):
        transition_rules (list[TransitionRule] | None):
        updated_at (datetime.datetime):
    """

    created_at: datetime.datetime
    id: str
    name: str
    source: str
    stages: list[WorkflowStage] | None
    transition_rules: list[TransitionRule] | None
    updated_at: datetime.datetime
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        id = self.id

        name = self.name

        source = self.source

        stages: list[dict[str, Any]] | None
        if isinstance(self.stages, list):
            stages = []
            for stages_type_0_item_data in self.stages:
                stages_type_0_item = stages_type_0_item_data.to_dict()
                stages.append(stages_type_0_item)

        else:
            stages = self.stages

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

        updated_at = self.updated_at.isoformat()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "createdAt": created_at,
                "id": id,
                "name": name,
                "source": source,
                "stages": stages,
                "transitionRules": transition_rules,
                "updatedAt": updated_at,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transition_rule import TransitionRule
        from ..models.workflow_stage import WorkflowStage

        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        id = d.pop("id")

        name = d.pop("name")

        source = d.pop("source")

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

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))

        workflow_template = cls(
            created_at=created_at,
            id=id,
            name=name,
            source=source,
            stages=stages,
            transition_rules=transition_rules,
            updated_at=updated_at,
        )

        workflow_template.additional_properties = d
        return workflow_template

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

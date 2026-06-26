from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

T = TypeVar("T", bound="WorkflowQuestionPolicy")


@_attrs_define
class WorkflowQuestionPolicy:
    """
    Attributes:
        answer_clears_awaiting_input_when_no_open_questions_remain (bool):
        enabled (bool):
        move_to_blocked (bool):
        sets_run_state (str):
    """

    answer_clears_awaiting_input_when_no_open_questions_remain: bool
    enabled: bool
    move_to_blocked: bool
    sets_run_state: str
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        answer_clears_awaiting_input_when_no_open_questions_remain = (
            self.answer_clears_awaiting_input_when_no_open_questions_remain
        )

        enabled = self.enabled

        move_to_blocked = self.move_to_blocked

        sets_run_state = self.sets_run_state

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "answerClearsAwaitingInputWhenNoOpenQuestionsRemain": answer_clears_awaiting_input_when_no_open_questions_remain,
                "enabled": enabled,
                "moveToBlocked": move_to_blocked,
                "setsRunState": sets_run_state,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        answer_clears_awaiting_input_when_no_open_questions_remain = d.pop(
            "answerClearsAwaitingInputWhenNoOpenQuestionsRemain"
        )

        enabled = d.pop("enabled")

        move_to_blocked = d.pop("moveToBlocked")

        sets_run_state = d.pop("setsRunState")

        workflow_question_policy = cls(
            answer_clears_awaiting_input_when_no_open_questions_remain=answer_clears_awaiting_input_when_no_open_questions_remain,
            enabled=enabled,
            move_to_blocked=move_to_blocked,
            sets_run_state=sets_run_state,
        )

        workflow_question_policy.additional_properties = d
        return workflow_question_policy

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

from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

T = TypeVar("T", bound="Question")


@_attrs_define
class Question:
    """
    Attributes:
        created_at (datetime.datetime):
        id (str):
        project_id (str):
        prompt (str):
        status (str):
        updated_at (datetime.datetime):
        work_item_id (str):
        actor (str | Unset):
        answer (str | Unset):
        answered_at (datetime.datetime | None | Unset):
        answered_by (str | Unset):
        pty_id (str | Unset):
        run_id (str | Unset):
        session_id (str | Unset):
    """

    created_at: datetime.datetime
    id: str
    project_id: str
    prompt: str
    status: str
    updated_at: datetime.datetime
    work_item_id: str
    actor: str | Unset = UNSET
    answer: str | Unset = UNSET
    answered_at: datetime.datetime | None | Unset = UNSET
    answered_by: str | Unset = UNSET
    pty_id: str | Unset = UNSET
    run_id: str | Unset = UNSET
    session_id: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        id = self.id

        project_id = self.project_id

        prompt = self.prompt

        status = self.status

        updated_at = self.updated_at.isoformat()

        work_item_id = self.work_item_id

        actor = self.actor

        answer = self.answer

        answered_at: None | str | Unset
        if isinstance(self.answered_at, Unset):
            answered_at = UNSET
        elif isinstance(self.answered_at, datetime.datetime):
            answered_at = self.answered_at.isoformat()
        else:
            answered_at = self.answered_at

        answered_by = self.answered_by

        pty_id = self.pty_id

        run_id = self.run_id

        session_id = self.session_id

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "createdAt": created_at,
                "id": id,
                "projectId": project_id,
                "prompt": prompt,
                "status": status,
                "updatedAt": updated_at,
                "workItemId": work_item_id,
            }
        )
        if actor is not UNSET:
            field_dict["actor"] = actor
        if answer is not UNSET:
            field_dict["answer"] = answer
        if answered_at is not UNSET:
            field_dict["answeredAt"] = answered_at
        if answered_by is not UNSET:
            field_dict["answeredBy"] = answered_by
        if pty_id is not UNSET:
            field_dict["ptyId"] = pty_id
        if run_id is not UNSET:
            field_dict["runId"] = run_id
        if session_id is not UNSET:
            field_dict["sessionId"] = session_id

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        id = d.pop("id")

        project_id = d.pop("projectId")

        prompt = d.pop("prompt")

        status = d.pop("status")

        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))

        work_item_id = d.pop("workItemId")

        actor = d.pop("actor", UNSET)

        answer = d.pop("answer", UNSET)

        def _parse_answered_at(data: object) -> datetime.datetime | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, str):
                    raise TypeError()
                answered_at_type_0 = datetime.datetime.fromisoformat(data)

                return answered_at_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(datetime.datetime | None | Unset, data)

        answered_at = _parse_answered_at(d.pop("answeredAt", UNSET))

        answered_by = d.pop("answeredBy", UNSET)

        pty_id = d.pop("ptyId", UNSET)

        run_id = d.pop("runId", UNSET)

        session_id = d.pop("sessionId", UNSET)

        question = cls(
            created_at=created_at,
            id=id,
            project_id=project_id,
            prompt=prompt,
            status=status,
            updated_at=updated_at,
            work_item_id=work_item_id,
            actor=actor,
            answer=answer,
            answered_at=answered_at,
            answered_by=answered_by,
            pty_id=pty_id,
            run_id=run_id,
            session_id=session_id,
        )

        question.additional_properties = d
        return question

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

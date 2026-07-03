from __future__ import annotations

import datetime
from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.mail_address import MailAddress
    from ..models.mail_recipient import MailRecipient


T = TypeVar("T", bound="MailMessage")


@_attrs_define
class MailMessage:
    """
    Attributes:
        created_at (datetime.datetime):
        from_ (MailAddress):
        id (str):
        priority (str):
        recipients (list[MailRecipient] | None):
        type_ (str):
        body (str | Unset):
        dispatch_id (str | Unset):
        payload (str | Unset):
        project_id (str | Unset):
        pty_id (str | Unset):
        reply_to_id (str | Unset):
        run_id (str | Unset):
        session_id (str | Unset):
        subject (str | Unset):
        thread_id (str | Unset):
        work_item_id (str | Unset):
    """

    created_at: datetime.datetime
    from_: MailAddress
    id: str
    priority: str
    recipients: list[MailRecipient] | None
    type_: str
    body: str | Unset = UNSET
    dispatch_id: str | Unset = UNSET
    payload: str | Unset = UNSET
    project_id: str | Unset = UNSET
    pty_id: str | Unset = UNSET
    reply_to_id: str | Unset = UNSET
    run_id: str | Unset = UNSET
    session_id: str | Unset = UNSET
    subject: str | Unset = UNSET
    thread_id: str | Unset = UNSET
    work_item_id: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        from_ = self.from_.to_dict()

        id = self.id

        priority = self.priority

        recipients: list[dict[str, Any]] | None
        if isinstance(self.recipients, list):
            recipients = []
            for recipients_type_0_item_data in self.recipients:
                recipients_type_0_item = recipients_type_0_item_data.to_dict()
                recipients.append(recipients_type_0_item)

        else:
            recipients = self.recipients

        type_ = self.type_

        body = self.body

        dispatch_id = self.dispatch_id

        payload = self.payload

        project_id = self.project_id

        pty_id = self.pty_id

        reply_to_id = self.reply_to_id

        run_id = self.run_id

        session_id = self.session_id

        subject = self.subject

        thread_id = self.thread_id

        work_item_id = self.work_item_id

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "createdAt": created_at,
                "from": from_,
                "id": id,
                "priority": priority,
                "recipients": recipients,
                "type": type_,
            }
        )
        if body is not UNSET:
            field_dict["body"] = body
        if dispatch_id is not UNSET:
            field_dict["dispatchId"] = dispatch_id
        if payload is not UNSET:
            field_dict["payload"] = payload
        if project_id is not UNSET:
            field_dict["projectId"] = project_id
        if pty_id is not UNSET:
            field_dict["ptyId"] = pty_id
        if reply_to_id is not UNSET:
            field_dict["replyToId"] = reply_to_id
        if run_id is not UNSET:
            field_dict["runId"] = run_id
        if session_id is not UNSET:
            field_dict["sessionId"] = session_id
        if subject is not UNSET:
            field_dict["subject"] = subject
        if thread_id is not UNSET:
            field_dict["threadId"] = thread_id
        if work_item_id is not UNSET:
            field_dict["workItemId"] = work_item_id

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.mail_address import MailAddress
        from ..models.mail_recipient import MailRecipient

        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))

        from_ = MailAddress.from_dict(d.pop("from"))

        id = d.pop("id")

        priority = d.pop("priority")

        def _parse_recipients(data: object) -> list[MailRecipient] | None:
            if data is None:
                return data
            try:
                if not isinstance(data, list):
                    raise TypeError()
                recipients_type_0 = []
                _recipients_type_0 = data
                for recipients_type_0_item_data in _recipients_type_0:
                    recipients_type_0_item = MailRecipient.from_dict(
                        recipients_type_0_item_data
                    )

                    recipients_type_0.append(recipients_type_0_item)

                return recipients_type_0
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(list[MailRecipient] | None, data)

        recipients = _parse_recipients(d.pop("recipients"))

        type_ = d.pop("type")

        body = d.pop("body", UNSET)

        dispatch_id = d.pop("dispatchId", UNSET)

        payload = d.pop("payload", UNSET)

        project_id = d.pop("projectId", UNSET)

        pty_id = d.pop("ptyId", UNSET)

        reply_to_id = d.pop("replyToId", UNSET)

        run_id = d.pop("runId", UNSET)

        session_id = d.pop("sessionId", UNSET)

        subject = d.pop("subject", UNSET)

        thread_id = d.pop("threadId", UNSET)

        work_item_id = d.pop("workItemId", UNSET)

        mail_message = cls(
            created_at=created_at,
            from_=from_,
            id=id,
            priority=priority,
            recipients=recipients,
            type_=type_,
            body=body,
            dispatch_id=dispatch_id,
            payload=payload,
            project_id=project_id,
            pty_id=pty_id,
            reply_to_id=reply_to_id,
            run_id=run_id,
            session_id=session_id,
            subject=subject,
            thread_id=thread_id,
            work_item_id=work_item_id,
        )

        mail_message.additional_properties = d
        return mail_message

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

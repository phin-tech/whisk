from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

T = TypeVar("T", bound="ClearDaemonResponse")


@_attrs_define
class ClearDaemonResponse:
    """
    Attributes:
        bookmarks_cleared (int):
        forwards_cleared (int):
        projects_cleared (int):
        ptys_cleared (int):
        sessions_cleared (int):
        work_items_cleared (int):
    """

    bookmarks_cleared: int
    forwards_cleared: int
    projects_cleared: int
    ptys_cleared: int
    sessions_cleared: int
    work_items_cleared: int
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        bookmarks_cleared = self.bookmarks_cleared

        forwards_cleared = self.forwards_cleared

        projects_cleared = self.projects_cleared

        ptys_cleared = self.ptys_cleared

        sessions_cleared = self.sessions_cleared

        work_items_cleared = self.work_items_cleared

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "bookmarksCleared": bookmarks_cleared,
                "forwardsCleared": forwards_cleared,
                "projectsCleared": projects_cleared,
                "ptysCleared": ptys_cleared,
                "sessionsCleared": sessions_cleared,
                "workItemsCleared": work_items_cleared,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        bookmarks_cleared = d.pop("bookmarksCleared")

        forwards_cleared = d.pop("forwardsCleared")

        projects_cleared = d.pop("projectsCleared")

        ptys_cleared = d.pop("ptysCleared")

        sessions_cleared = d.pop("sessionsCleared")

        work_items_cleared = d.pop("workItemsCleared")

        clear_daemon_response = cls(
            bookmarks_cleared=bookmarks_cleared,
            forwards_cleared=forwards_cleared,
            projects_cleared=projects_cleared,
            ptys_cleared=ptys_cleared,
            sessions_cleared=sessions_cleared,
            work_items_cleared=work_items_cleared,
        )

        clear_daemon_response.additional_properties = d
        return clear_daemon_response

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

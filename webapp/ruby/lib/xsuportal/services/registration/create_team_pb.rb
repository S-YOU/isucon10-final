# Generated by the protocol buffer compiler.  DO NOT EDIT!
# source: xsuportal/services/registration/create_team.proto

require 'google/protobuf'

Google::Protobuf::DescriptorPool.generated_pool.build do
  add_file("xsuportal/services/registration/create_team.proto", :syntax => :proto3) do
    add_message "xsuportal.proto.services.registration.CreateTeamRequest" do
      optional :team_name, :string, 1
      optional :name, :string, 2
      optional :email_address, :string, 3
      optional :is_student, :bool, 4
    end
    add_message "xsuportal.proto.services.registration.CreateTeamResponse" do
      optional :team_id, :int64, 1
    end
  end
end

module Xsuportal
  module Proto
    module Services
      module Registration
        CreateTeamRequest = ::Google::Protobuf::DescriptorPool.generated_pool.lookup("xsuportal.proto.services.registration.CreateTeamRequest").msgclass
        CreateTeamResponse = ::Google::Protobuf::DescriptorPool.generated_pool.lookup("xsuportal.proto.services.registration.CreateTeamResponse").msgclass
      end
    end
  end
end
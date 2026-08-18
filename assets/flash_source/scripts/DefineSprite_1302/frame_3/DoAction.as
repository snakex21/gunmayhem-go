function slideback()
{
   if(_parent.menu1.slide == 2)
   {
      _parent.menu1.slide = 1;
   }
   if(_parent.menu2.slide == 2)
   {
      _parent.menu2.slide = 1;
   }
   if(_parent.menu3.slide == 2)
   {
      _parent.menu3.slide = 1;
   }
   if(_parent.menu4.slide == 2)
   {
      _parent.menu4.slide = 1;
   }
}
function update()
{
   player.head.gotoAndStop(colornumber + 1);
   player.body.gotoAndStop(colornumber + 1);
   player.leg1.leg.gotoAndStop(colornumber + 1);
   player.leg2.leg.gotoAndStop(colornumber + 1);
   player.hand2.hand.gotoAndStop(colornumber + 1);
   player.gundisplayhand.gotoAndStop(colornumber + 1);
}
switch(_name)
{
   case "menu1":
      colornumber = _parent.savedata.data.p1color;
      inputname.text = _parent.savedata.data.p1name;
      player.shirt.gotoAndStop(_parent.savedata.data.p1shirt);
      player.hat.gotoAndStop(_parent.savedata.data.p1hat);
      player.eyes.gotoAndStop(_parent.savedata.data.p1hat);
      player.gundisplay.gotoAndStop(_parent.savedata.data.p1gun);
      perkdisplay.gotoAndStop(_parent.savedata.data.p1perk);
      playertype = _parent.savedata.data.p1ptype;
      teamselect.team = _parent.savedata.data.p1team;
      break;
   case "menu2":
      colornumber = _parent.savedata.data.p2color;
      inputname.text = _parent.savedata.data.p2name;
      player.shirt.gotoAndStop(_parent.savedata.data.p2shirt);
      player.hat.gotoAndStop(_parent.savedata.data.p2hat);
      player.eyes.gotoAndStop(_parent.savedata.data.p2hat);
      player.gundisplay.gotoAndStop(_parent.savedata.data.p2gun);
      perkdisplay.gotoAndStop(_parent.savedata.data.p2perk);
      playertype = _parent.savedata.data.p2ptype;
      teamselect.team = _parent.savedata.data.p2team;
      break;
   case "menu3":
      colornumber = _parent.savedata.data.p3color;
      inputname.text = _parent.savedata.data.p3name;
      player.shirt.gotoAndStop(_parent.savedata.data.p3shirt);
      player.hat.gotoAndStop(_parent.savedata.data.p3hat);
      player.eyes.gotoAndStop(_parent.savedata.data.p3hat);
      player.gundisplay.gotoAndStop(_parent.savedata.data.p3gun);
      perkdisplay.gotoAndStop(_parent.savedata.data.p3perk);
      playertype = _parent.savedata.data.p3ptype;
      teamselect.team = _parent.savedata.data.p3team;
      break;
   case "menu4":
      colornumber = _parent.savedata.data.p4color;
      inputname.text = _parent.savedata.data.p4name;
      player.shirt.gotoAndStop(_parent.savedata.data.p4shirt);
      player.hat.gotoAndStop(_parent.savedata.data.p4hat);
      player.eyes.gotoAndStop(_parent.savedata.data.p4hat);
      player.gundisplay.gotoAndStop(_parent.savedata.data.p4gun);
      perkdisplay.gotoAndStop(_parent.savedata.data.p4perk);
      playertype = _parent.savedata.data.p4ptype;
      teamselect.team = _parent.savedata.data.p4team;
}
updated = false;
updatetime = 1;
originy = 100;
ai_notice._alpha = 0;
if(playertype == 0)
{
   slide = 0;
   _Y = 400;
}
else if(playertype == 1)
{
   slide = 1;
   _Y = 0;
}
else if(playertype == 2)
{
   slide = 1;
   _Y = 0;
   ai_notice._alpha = 100;
}
this.onEnterFrame = function()
{
   if(!updated)
   {
      update();
      updated = true;
   }
   if(updatetime != 0)
   {
      updatetime += 1;
   }
   if(updatetime == 2)
   {
      updatetime = 0;
   }
   if(slide == 0)
   {
      _Y = _Y + (originy + 400 - _Y) / 2;
   }
   else if(slide == 1)
   {
      _Y = _Y + (originy - _Y) / 2;
   }
   else if(slide == 2)
   {
      _Y = _Y + (originy - 400 - _Y) / 2;
   }
   if(_Y % 1 != 0)
   {
      _Y = Math.round(_Y);
   }
   if(player.hat.getDepth() < player.eyes.getDepth() && player.eyes._currentframe != 8)
   {
      player.hat.swapDepths(player.eyes);
   }
};
btn_edit1.onRelease = function()
{
   slideback();
   slide = 2;
   selectionpanel.gotoAndStop(1);
   displaytext.text = "";
};
btn_edit2.onRelease = function()
{
   slideback();
   slide = 2;
   if(player.hat._currentframe > 12)
   {
      selectionpanel.gotoAndStop(5);
   }
   else
   {
      selectionpanel.gotoAndStop(2);
   }
   displaytext.text = "";
};
btn_edit3.onRelease = function()
{
   slideback();
   slide = 2;
   selectionpanel.gotoAndStop(3);
   displaytext.text = "";
};
btn_edit4.onRelease = function()
{
   slideback();
   slide = 2;
   selectionpanel.gotoAndStop(4);
   displaytext.text = "";
};
btn_back.onRelease = function()
{
   slideback();
   slide = 1;
};
btn_clear.onRelease = function()
{
   slide = 0;
   playertype = 0;
};
btn_player_human.onRelease = function()
{
   slide = 1;
   playertype = 1;
   ai_notice._alpha = 0;
};
btn_player_ai.onRelease = function()
{
   slide = 1;
   playertype = 2;
   ai_notice._alpha = 100;
};
